// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

declare module "xxhashjs" {
  interface XXHash {
    h64(value: string, seed?: number): { toString(): string };
  }

  const XXH: XXHash;
  export default XXH;
}
