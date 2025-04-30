declare module "bs58" {
  function encode(buffer: Uint8Array | Buffer): string;
  function decode(str: string): Buffer;
  export default {
    encode,
    decode,
  };
}
