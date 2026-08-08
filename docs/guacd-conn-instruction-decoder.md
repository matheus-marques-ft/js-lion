# Guacd Instruction Decoder

## Protocol Format

A Guacamole instruction consists of an opcode and zero or more arguments. Each
element uses the `LENGTH.VALUE` format, elements are separated by commas, and
the whole instruction ends with a semicolon.

`LENGTH` denotes the number of Unicode code points in `VALUE`, not the number
of UTF-8 bytes. Commas and semicolons may appear inside element content —
only after the declared number of code points has been read does a following
comma or semicolon act as a structural delimiter.

## InstructionDecoder

`InstructionDecoder` reads instructions incrementally from an `io.Reader`.
Each call to `ReadInstruction()` returns exactly one complete `Instruction`,
and retains in the buffer whatever data belongs to the next instruction.

When creating a decoder:

- If the input is already a `*bufio.Reader`, it is reused directly;
- Any other `io.Reader` is wrapped in a new `bufio.Reader`;
- For an unbuffered reader, the same decoder should be reused continuously,
  to avoid losing any pre-read data;
- The decoder is not intended for concurrent reads.

The process for reading a single instruction is as follows:

1. Read a non-empty ASCII decimal length prefix;
2. Read the declared number of UTF-8-encoded Unicode code points;
3. Verify that the character following the element is a comma or semicolon;
4. On a comma, continue reading the next element; on a semicolon, return the
   instruction.

## Parsing Limits

The decoder enforces the following fixed limits:

| Limit | Value |
| --- | ---: |
| Max Unicode code points per element | 8192 |
| Max digits in the length prefix | 5 |
| Max elements per instruction (including the opcode) | 128 |
| Max cumulative bytes per instruction | 32768 |

The cumulative byte count includes the length prefix, element content, and
delimiters. Exceeding any of these limits immediately aborts parsing of the
current instruction.

## Error Semantics

Parsing errors are classified via error values defined within the package:

| Error | Meaning |
| --- | --- |
| `ErrInstructionMissSemicolon` | The string being parsed did not end with a semicolon |
| `ErrInstructionMissDot` | Missing dot after the length prefix |
| `ErrInstructionBadDigit` | Length is empty or contains non-digit characters |
| `ErrInstructionBadContent` | Element content is incomplete |
| `ErrInstructionBadTerminator` | The character following an element is neither a comma nor a semicolon |
| `ErrInstructionTooLong` | The length prefix, element length, or cumulative byte count exceeds the limit |
| `ErrInstructionTooManyElements` | The instruction has too many elements |
| `ErrInstructionInvalidUTF8` | An element contains invalid UTF-8 |
| `ErrInstructionTrailingData` | Data remains after a single-instruction string |

If the stream ends before any bytes of a new instruction have been read,
`io.EOF` is returned. If EOF occurs partway through reading the length,
content, or terminator, the returned error can be identified via
`errors.Is(err, io.ErrUnexpectedEOF)`, while still retaining the
corresponding parse-error classification. Non-EOF underlying read errors,
such as network timeouts, are returned as-is.

When a parse error occurs, the input already consumed for the current
instruction is not rewound — the caller should terminate the current stream
or connection.

## String Parsing and Serialization

`ParseInstructionString()` uses `InstructionDecoder` to parse the input, and
requires the input to contain exactly one complete instruction:

- The input must end with a semicolon;
- EOF must be reached immediately after the first instruction;
- Multiple instructions, or any trailing content, result in
  `ErrInstructionTrailingData`.

`Instruction.String()` generates length prefixes for the opcode and all
arguments uniformly, based on the Unicode code point count. The generated
result is cached in `ProtocolForm`, and subsequent calls return the cached
content directly.

## Tunnel Integration

`Tunnel` creates a decoder based on its `bufio.Reader` when the connection is
established, and reuses it for the lifetime of the connection.
`Tunnel.ReadInstruction()` sets a 15-second read deadline once before each
instruction begins reading, after which the decoder reads and returns one
instruction.

`Tunnel.Read()` reuses `ReadInstruction()` and re-serializes the parsed
instruction back into bytes. The `expect()` call during the handshake phase
also validates the expected opcode through this same read path.

## Replay File Integration

The replay file reading function accepts an existing `*bufio.Reader` and
reads instructions one at a time via `InstructionDecoder`. Since the decoder
reuses that reader directly, repeated calls do not lose subsequent data
already buffered in the reader. Replay time scanning traverses instructions
on this basis and extracts `sync` timestamps.
