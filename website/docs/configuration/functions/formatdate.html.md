---
layout: "functions"
page_title: "formatdate - Functions - Configuration Language"
sidebar_current: "docs-funcs-datetime-formatdate"
description: |-
  The formatdate function converts a timestamp into a different time format.
---

# `formatdate` Function

-> **Note:** This page is about Terraform 0.12 and later. For Terraform 0.11 and
earlier, see
[0.11 Configuration Language: Interpolation Syntax](../../configuration-0-11/interpolation.html).

`formatdate` converts a timestamp into a different time format.

```hcl
formatdate(spec, timestamp)
```

In the Terraform language, timestamps are conventionally represented as
strings using [RFC 3339](https://tools.ietf.org/html/rfc3339)
"Date and Time format" syntax. `formatdate` requires the `timestamp` argument
to be a string conforming to this syntax.

## Examples

```
> formatdate("DD MMM YYYY hh:mm ZZZ", "2018-01-02T23:12:01Z")
02 Jan 2018 23:12 UTC
> formatdate("EEEE, DD-MMM-YY hh:mm:ss ZZZ", "2018-01-02T23:12:01Z")
Tuesday, 02-Jan-18 23:12:01 UTC
> formatdate("EEE, DD MMM YYYY hh:mm:ss ZZZ", "2018-01-02T23:12:01-08:00")
Tue, 02 Jan 2018 23:12:01 -0800
> formatdate("MMM DD, YYYY", "2018-01-02T23:12:01Z")
Jan 02, 2018
> formatdate("HH:mmaa", "2018-01-02T23:12:01Z")
11:12pm
```

## Specification Syntax

The format specification is a string that includes formatting sequences from
the following table. This function is intended for producing common
_machine-oriented_ timestamp formats such as those defined in RFC822, RFC850,
and RFC1123. It is not suitable for truly human-oriented date formatting
because it is not locale-aware. In particular, it can produce month and day
names only in English.

The specification may contain the following sequences:

| Sequence  | Result                                                                   |
| --------- | ------------------------------------------------------------------------ |
| `YYYY`    | Four (or more) digit year, like "2006".                                  |
| `YY`      | The year modulo 100, zero padded to at least two digits, like "06".      |
| `MMMM`    | English month name unabbreviated, like "January".                        |
| `MMM`     | English month name abbreviated to three letters, like "Jan".             |
| `MM`      | Month number zero-padded to two digits, like "01" for January.           |
| `M`       | Month number with no padding, like "1" for January.                      |
| `DD`      | Day of month number zero-padded to two digits, like "02".                |
| `D`       | Day of month number with no padding, like "2".                           |
| `EEEE`    | English day of week name unabbreviated, like "Monday".                   |
| `EEE`     | English day of week name abbreviated to three letters, like "Mon".       |
| `hh`      | 24-hour number zero-padded to two digits, like "02".                     |
| `h`       | 24-hour number unpadded, like "2".                                       |
| `HH`      | 12-hour number zero-padded to two digits, like "02".                     |
| `H`       | 12-hour number unpadded, like "2".                                       |
| `AA`      | Hour AM/PM marker in uppercase, like "AM".                               |
| `aa`      | Hour AM/PM marker in lowercase, like "am".                               |
| `mm`      | Minute within hour zero-padded to two digits, like "05".                 |
| `m`       | Minute within hour unpadded, like "5".                                   |
| `ss`      | Second within minute zero-padded to two digits, like "09".               |
| `s`       | Second within minute, like "9".                                          |
| `ZZZZZ`   | Timezone offset with colon separating hours and minutes, like "-08:00".  |
| `ZZZZ`    | Timezone offset with just sign and digit, like "-0800".                  |
| `ZZZ`     | Like `ZZZZ` but with a special case "UTC" for UTC.                       |
| `Z`       | Like `ZZZZZ` but with a special case "Z" for UTC.                        |

Any non-letter characters, such as punctuation, are reproduced verbatim in the
output. To include literal letters in the format string, enclose them in single
quotes `'`. To include a literal quote, escape it by doubling the quotes.

```
> formatdate("h'h'mm", "2018-01-02T23:12:01-08:00")
23h12
> formatdate("H 'o''clock'", "2018-01-02T23:12:01-08:00")
11 o'clock
```

This format specification syntax is intended to make it easy for a reader
to guess which format will result even if they are not experts on the syntax.
Therefore there are no predefined shorthands for common formats, but format
strings for various RFC-specified formats are given below to be copied into your
configuration as needed:

- [RFC 822](https://tools.ietf.org/html/rfc822#section-5) and
  [RFC RFC 2822](https://tools.ietf.org/html/rfc2822#section-3.3):
  `"DD MMM YYYY hh:mm ZZZ"`
- [RFC 850](https://tools.ietf.org/html/rfc850#section-2.1.4):
  `"EEEE, DD-MMM-YY hh:mm:ss ZZZ"`
- [RFC 1123](https://tools.ietf.org/html/rfc1123#section-5.2.14):
  `"EEE, DD MMM YYYY hh:mm:ss ZZZ"`
- [RFC 3339](https://tools.ietf.org/html/rfc3339):
  `"YYYY-MM-DD'T'hh:mm:ssZ"` (but this is also the input format, so such a
  conversion is redundant.)

## Related Functions

* [`format`](./format.html) is a more general formatting function for arbitrary
  data.
* [`timestamp`](./timestamp.html) returns the current date and time in a format
  suitable for input to `formatdate`.
