# Stream Logic Gaps

- Fixed: existing-by-name updates now target the resolved stream ID instead of a blank spec ID.
- Fixed: `FAILED` and `DELETED` streams no longer report terminal success.
- Fixed: `CREATING`, `UPDATING`, and `DELETING` now requeue instead of silently stalling.
- Fixed: delete resolves IDs from spec, status, or name lookup and only completes once the stream is gone.
- Fixed: secret generation now fails safely when `MessagesEndpoint` is missing instead of panicking.
