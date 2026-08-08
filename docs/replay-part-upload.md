# Recording Segment Upload Change Log

## Pre-upload Precheck

Before generating the compressed archive and starting the upload, the
uploader performs a precheck on all of the raw segments belonging to the
current recording:

- Only segments whose names match `{session_id}.{index}.part` for the
  current session are collected;
- They are sorted by numeric segment index;
- Segment indices must start at 0 and be contiguous;
- Each segment is scanned in full using the same `InstructionDecoder`;
- The end position of each `sync` is computed by subtracting the buffer's
  pre-read bytes from the cumulative bytes read, without calling `Seek` for
  every `sync`;
- Each segment must contain at least one `sync` instruction with a valid
  timestamp.

If any segment has an invalid instruction, no valid `sync`, an invalid
`sync` timestamp, or a non-contiguous segment index, the entire recording is
not uploaded, and the original files are kept for manual handling.

## Last-Segment Recovery

Automatic recovery applies only to the last segment, and requires that all
of the following conditions hold:

- The parse error can be identified via
  `errors.Is(err, io.ErrUnexpectedEOF)`;
- At least one complete `sync` with a valid timestamp exists before the
  error.

When these conditions are met, the last segment is truncated to the end
position of the last complete `sync`, and then rescanned. If the rescan
still produces an error, the entire recording is not uploaded. Non-final
segments never undergo automatic truncation.

## Metadata and Upload

Once all segments have passed the precheck:

1. Each segment's start time, end time, duration, and file size are
   regenerated based on the scan results;
2. The corresponding `.part.meta` file is rewritten;
3. The staging directory for this upload is cleared and recreated;
4. All segments are compressed and the replay metadata is generated;
5. The upload begins.

Validation failure occurs before compression and upload, so no faulty
segment is skipped in order to continue uploading the remaining segments.
