# bitpack Roadmap

This document tracks planned features and future directions for the library.

---

## v0.2 — Quality of life

- [ ] **Streaming encoder/decoder** — `bitpack.NewReader(r io.Reader)` / `bitpack.NewWriter(w io.Writer)` for incremental processing of large binary streams without loading the entire payload into memory.
- [ ] **Nested structs** — allow embedding another bitpack struct as a field, so protocol layers can be composed naturally (e.g. an IP header containing a TCP header).
- [ ] **Array fields** — `[N]T` field support with a single `bits:"W"` tag applying to each element.
- [ ] **Skip/padding fields** — `bits:"-"` or a blank identifier field to skip reserved/padding bits without consuming a named field.

## v0.3 — Schema DSL

- [ ] **`.bitpack` schema file format** — a human-editable text DSL that describes a packed format without requiring Go source:
  ```
  struct TCPFlags {
      CWR:  1 bool
      ECE:  1 bool
      ACK:  1 bool
      SYN:  1 bool
      FIN:  1 bool
  }
  ```
- [ ] **Schema registry** — load `.bitpack` files at runtime and marshal/unmarshal into `map[string]any` for dynamic use cases.
- [ ] **Schema validation** — detect overlapping fields, total-bits mismatches, and type incompatibilities at schema load time.

## v0.4 — Code generation

- [ ] **Go codegen** — `bitpackgen` CLI that reads `.bitpack` schema files and emits optimised, zero-reflection Go code (faster than the reflect-based runtime).
- [ ] **TypeScript codegen** — emit TypeScript interfaces and encode/decode functions for use in browser or Node.js tooling.
- [ ] **Rust codegen** — emit `#[repr(packed)]` Rust structs with matching `From<&[u8]>` implementations.
- [ ] **C codegen** — emit portable C structs with bitfield macros compatible with GCC/Clang.

## v0.5 — Visual inspector

- [ ] **Browser-based visual inspector** — a web UI where you paste hex bytes and a schema, and see an interactive colour-coded breakdown of every bit.  Can be self-hosted or run as a static page.
- [ ] **VSCode extension** — hover over a hex literal in Go source and see a rendered bitpack breakdown inline.

## v1.0 — Built-in protocol schemas

Ship a `protocols/` package with ready-to-use schemas for common binary formats:

- [ ] **TCP** — header flags, options
- [ ] **UDP** — header
- [ ] **IPv4 / IPv6** — headers
- [ ] **DNS** — message header, flags
- [ ] **BLE (Bluetooth Low Energy)** — advertisement packets, GATT attributes
- [ ] **CAN bus** — standard and extended frames
- [ ] **MQTT** — fixed-header flags
- [ ] **IEEE 802.11 (Wi-Fi)** — frame control field

## Long-term / research

- [ ] **Fuzzing harness** — `go-fuzz` / `testing.F` corpus for every built-in protocol schema.
- [ ] **Benchmarks** — systematic micro-benchmarks comparing reflect-based vs codegen paths.
- [ ] **`#nosplit` / unsafe fast path** — optional unsafe read/write path for hot loops (e.g. packet capture at line rate).
- [ ] **WASM build** — compile the schema interpreter to WebAssembly for use in the browser inspector.
