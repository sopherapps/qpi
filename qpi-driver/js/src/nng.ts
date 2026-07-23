/**
 * A minimal implementation of the nanomsg Scalability Protocol (SP) pipeline
 * (PUSH/PULL) over a TLS socket — enough for a driver to talk to QPI-UI's NNG
 * endpoints without any native NNG binding, keeping this SDK dependency-free
 * like the QPI JS client.
 *
 * The server speaks SP over `tls+tcp` via mangos. Interop needs only two
 * things (mangos `transport/tcp` + SP handshake):
 *
 *   1. On connect, each peer sends an 8-byte header
 *      `0x00 'S' 'P' 0x00 <proto:uint16be> 0x00 0x00`, then reads and
 *      validates the peer's. PUSH is protocol 80, PULL is 81; a PUSH may only
 *      talk to a PULL and vice-versa.
 *   2. Each subsequent message is framed as an 8-byte big-endian length
 *      followed by that many payload bytes.
 */

import * as tls from "node:tls";

const SP_VERSION = 0x00;
export const PROTO_PUSH = 80;
export const PROTO_PULL = 81;
const HEADER_SIZE = 8;
const LENGTH_PREFIX_SIZE = 8;

/** The pipeline role this end plays: a source (PUSH) or a sink (PULL). */
export type Role = "push" | "pull";

/**
 * Build the 8-byte SP handshake header for a protocol number:
 * `0x00 'S' 'P' 0x00 <proto:uint16be> 0x00 0x00`.
 */
export function buildSpHeader(proto: number): Buffer {
  return Buffer.from([
    0x00,
    0x53, // 'S'
    0x50, // 'P'
    SP_VERSION,
    (proto >> 8) & 0xff,
    proto & 0xff,
    0x00,
    0x00,
  ]);
}

/** Validate an SP header's magic and return the peer's protocol number. */
export function parseSpHeader(header: Buffer): number {
  const magicOk =
    header.length >= HEADER_SIZE &&
    header[0] === 0x00 &&
    header[1] === 0x53 &&
    header[2] === 0x50 &&
    header[3] === SP_VERSION;
  if (!magicOk) {
    throw new Error("qpi-driver: invalid SP handshake header from peer");
  }
  return (header[4] << 8) | header[5];
}

/** Frame a payload as an 8-byte big-endian length prefix followed by the bytes. */
export function frame(payload: Buffer): Buffer {
  const prefix = Buffer.alloc(LENGTH_PREFIX_SIZE);
  prefix.writeBigUInt64BE(BigInt(payload.length));
  return Buffer.concat([prefix, payload]);
}

/**
 * Accumulates received bytes and yields complete framed messages. Handles TCP
 * segmentation: a message split across chunks, or several messages in one.
 */
export class MessageAssembler {
  private buffer: Buffer = Buffer.alloc(0);

  push(chunk: Buffer): void {
    this.buffer = Buffer.concat([this.buffer, chunk]);
  }

  /** Take the bytes remaining after any consumed messages (e.g. a handshake). */
  consume(n: number): void {
    this.buffer = this.buffer.subarray(n);
  }

  get length(): number {
    return this.buffer.length;
  }

  peek(n: number): Buffer {
    return this.buffer.subarray(0, n);
  }

  /** Pull every complete message currently buffered. */
  drain(): Buffer[] {
    const messages: Buffer[] = [];
    for (;;) {
      if (this.buffer.length < LENGTH_PREFIX_SIZE) {
        break;
      }
      const length = Number(this.buffer.readBigUInt64BE(0));
      if (this.buffer.length < LENGTH_PREFIX_SIZE + length) {
        break;
      }
      messages.push(
        this.buffer.subarray(LENGTH_PREFIX_SIZE, LENGTH_PREFIX_SIZE + length),
      );
      this.buffer = this.buffer.subarray(LENGTH_PREFIX_SIZE + length);
    }
    return messages;
  }
}

/** Coordinates for dialing a QPI-UI NNG endpoint over TLS. */
export interface DialOptions {
  host: string;
  port: number;
  /** PEM-encoded root CA to pin the server certificate against. */
  ca: string;
  /** Server name for SNI and certificate validation. */
  servername: string;
}

/**
 * One SP pipeline connection over TLS. A `push` socket sends; a `pull` socket
 * receives. Both complete the SP handshake on dial.
 */
export class PipelineSocket {
  private socket?: tls.TLSSocket;
  private readonly selfProto: number;
  private readonly peerProto: number;
  private readonly assembler = new MessageAssembler();
  private handshakeDone = false;
  private messageHandler: (data: Buffer) => void = () => {};
  private closed = false;

  constructor(private readonly role: Role) {
    this.selfProto = role === "push" ? PROTO_PUSH : PROTO_PULL;
    this.peerProto = role === "push" ? PROTO_PULL : PROTO_PUSH;
  }

  /** Connect, exchange SP headers, and resolve once the handshake completes. */
  dial(opts: DialOptions): Promise<void> {
    return new Promise((resolve, reject) => {
      let settled = false;
      const socket = tls.connect(
        {
          host: opts.host,
          port: opts.port,
          ca: opts.ca,
          servername: opts.servername,
          minVersion: "TLSv1.2",
        },
        () => {
          socket.write(buildSpHeader(this.selfProto));
        },
      );
      this.socket = socket;

      socket.on("data", (chunk: Buffer) => {
        this.assembler.push(chunk);
        if (!this.handshakeDone) {
          if (this.assembler.length < HEADER_SIZE) {
            return;
          }
          try {
            const peer = parseSpHeader(this.assembler.peek(HEADER_SIZE));
            if (peer !== this.peerProto) {
              throw new Error(
                `qpi-driver: SP protocol mismatch: expected peer ${this.peerProto}, got ${peer}`,
              );
            }
          } catch (err) {
            socket.destroy();
            if (!settled) {
              settled = true;
              reject(err);
            }
            return;
          }
          this.assembler.consume(HEADER_SIZE);
          this.handshakeDone = true;
          if (!settled) {
            settled = true;
            resolve();
          }
        }
        this.deliver();
      });

      socket.on("error", (err: Error) => {
        if (!settled) {
          settled = true;
          reject(err);
        }
      });
    });
  }

  /** Frame and send a message (length prefix + payload). */
  send(data: Buffer): void {
    if (!this.socket || this.closed) {
      throw new Error("qpi-driver: cannot send on a closed socket");
    }
    this.socket.write(frame(data));
  }

  /** Register the callback invoked once per received message. */
  onMessage(handler: (data: Buffer) => void): void {
    this.messageHandler = handler;
    if (this.handshakeDone) {
      this.deliver();
    }
  }

  /** Close the underlying socket. */
  close(): void {
    this.closed = true;
    this.socket?.destroy();
  }

  private deliver(): void {
    for (const message of this.assembler.drain()) {
      this.messageHandler(message);
    }
  }
}
