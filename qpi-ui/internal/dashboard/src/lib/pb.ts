import PocketBase from "pocketbase";

// Resolve local dev server to 8090, else use current browser origin
export const pb = new PocketBase(
  import.meta.env.DEV ? "http://localhost:8090" : undefined
);
