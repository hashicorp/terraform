## v4

- Encode, Decode, Marshal, and Unmarshal are changed to accept single argument. EncodeMulti and DecodeMulti are added as replacement.
- Added EncodeInt8/16/32/64 and EncodeUint8/16/32/64.
- Encoder changed to preserve type of numbers instead of chosing most compact encoding. The old behavior can be achieved with Encoder.UseCompactEncoding.

## v3.3

- `msgpack:",inline"` tag is restored to force inlining structs.

## v3.2

- Decoding extension types returns pointer to the value instead of the value. Fixes #153

## v3

- gopkg.in is not supported any more. Update import path to github.com/vmihailenco/msgpack.
- Msgpack maps are decoded into map[string]interface{} by default.
- EncodeSliceLen is removed in favor of EncodeArrayLen. DecodeSliceLen is removed in favor of DecodeArrayLen.
- Embedded structs are automatically inlined where possible.
- Time is encoded using extension as described in https://github.com/msgpack/msgpack/pull/209. Old format is supported as well.
- EncodeInt8/16/32/64 is replaced with EncodeInt. EncodeUint8/16/32/64 is replaced with EncodeUint. There should be no performance differences.
- DecodeInterface can now return int8/16/32 and uint8/16/32.
- PeekCode returns codes.Code instead of byte.
