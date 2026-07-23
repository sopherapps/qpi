/**
 * Root CA pinning helpers shared by the driver handshake.
 *
 * A driver downloads the server root CA over plain HTTP and pins it: the
 * SHA-256 of the certificate's DER bytes must match the fingerprint the
 * operator was given out of band, matching the Python and Go SDKs.
 */

import { createHash } from "node:crypto";

/** Extract the DER bytes of the first certificate in a PEM string. */
export function pemToDer(pem: string): Buffer {
  const match = pem.match(
    /-----BEGIN CERTIFICATE-----([\s\S]*?)-----END CERTIFICATE-----/,
  );
  if (!match) {
    throw new Error("qpi-driver: root CA is not valid PEM");
  }
  return Buffer.from(match[1].replace(/\s+/g, ""), "base64");
}

/** Return the hex SHA-256 fingerprint of a PEM certificate's DER bytes. */
export function fingerprintOf(pem: string): string {
  return createHash("sha256").update(pemToDer(pem)).digest("hex");
}

/**
 * Verify a downloaded PEM certificate against the pinned fingerprint. An empty
 * expected fingerprint skips the check; a mismatch throws.
 */
export function verifyFingerprint(pem: string, expected: string): void {
  if (!expected) {
    return;
  }
  const got = fingerprintOf(pem);
  if (got.toLowerCase() !== expected.toLowerCase()) {
    throw new Error(
      `qpi-driver: CRITICAL SECURITY ERROR: downloaded CA fingerprint (${got}) ` +
        `does not match the expected value (${expected}); ` +
        `the download channel may be compromised`,
    );
  }
}
