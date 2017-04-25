---
layout: "aws"
page_title: "AWS: aws_elastictranscoder_preset"
sidebar_current: "docs-aws-resource-elastic-transcoder-preset"
description: |-
  Provides an Elastic Transcoder preset resource.
---

# aws\_elastictranscoder\_preset

Provides an Elastic Transcoder preset resource.

## Example Usage

```hcl
resource "aws_elastictranscoder_preset" "bar" {
  container   = "mp4"
  description = "Sample Preset"
  name        = "sample_preset"

  audio = {
    audio_packing_mode = "SingleTrack"
    bit_rate           = 96
    channels           = 2
    codec              = "AAC"
    sample_rate        = 44100
  }

  audio_codec_options = {
    profile = "AAC-LC"
  }

  video = {
    bit_rate             = "1600"
    codec                = "H.264"
    display_aspect_ratio = "16:9"
    fixed_gop            = "false"
    frame_rate           = "auto"
    max_frame_rate       = "60"
    keyframes_max_dist   = 240
    max_height           = "auto"
    max_width            = "auto"
    padding_policy       = "Pad"
    sizing_policy        = "Fit"
  }

  video_codec_options = {
    Profile                  = "main"
    Level                    = "2.2"
    MaxReferenceFrames       = 3
    InterlaceMode            = "Progressive"
    ColorSpaceConversionMode = "None"
  }

  video_watermarks = {
    id                = "Terraform Test"
    max_width         = "20%"
    max_height        = "20%"
    sizing_policy     = "ShrinkToFit"
    horizontal_align  = "Right"
    horizontal_offset = "10px"
    vertical_align    = "Bottom"
    vertical_offset   = "10px"
    opacity           = "55.5"
    target            = "Content"
  }

  thumbnails = {
    format         = "png"
    interval       = 120
    max_width      = "auto"
    max_height     = "auto"
    padding_policy = "Pad"
    sizing_policy  = "Fit"
  }
}
```

## Argument Reference

See ["Create Preset"](http://docs.aws.amazon.com/elastictranscoder/latest/developerguide/create-preset.html) in the AWS docs for reference.

The following arguments are supported:

* `audio` - (Optional, Forces new resource) Audio parameters object (documented below).
* `audio_codec_options` - (Optional, Forces new resource) Codec options for the audio parameters (documented below)
* `container` - (Required, Forces new resource) The container type for the output file. Valid values are `flac`, `flv`, `fmp4`, `gif`, `mp3`, `mp4`, `mpg`, `mxf`, `oga`, `ogg`, `ts`, and `webm`.
* `description` - (Optional, Forces new resource) A description of the preset (maximum 255 characters)
* `name` - (Optional, Forces new resource) The name of the preset. (maximum 40 characters)
* `thumbnails` - (Optional, Forces new resource) Thumbnail parameters object (documented below)
* `video` - (Optional, Forces new resource) Video parameters object (documented below)
* `video_watermarks` - (Optional, Forces new resource) Watermark parameters for the video parameters (documented below)
* `video_codec_options` (Optional, Forces new resource) Codec options for the video parameters

The `audio` object supports the following:

* `audio_packing_mode` - The method of organizing audio channels and tracks. Use Audio:Channels to specify the number of channels in your output, and Audio:AudioPackingMode to specify the number of tracks and their relation to the channels. If you do not specify an Audio:AudioPackingMode, Elastic Transcoder uses SingleTrack.
* `bit_rate` - The bit rate of the audio stream in the output file, in kilobits/second. Enter an integer between 64 and 320, inclusive.
* `channels` - The number of audio channels in the output file
* `codec` - The audio codec for the output file. Valid values are `AAC`, `flac`, `mp2`, `mp3`, `pcm`, and `vorbis`.
* `sample_rate` - The sample rate of the audio stream in the output file, in hertz. Valid values are: `auto`, `22050`, `32000`, `44100`, `48000`, `96000`

The `audio_codec_options` object supports the following:

* `bit_depth` - The bit depth of a sample is how many bits of information are included in the audio samples. Valid values are `16` and `24`. (FLAC/PCM Only)
* `bit_order` - The order the bits of a PCM sample are stored in. The supported value is LittleEndian. (PCM Only)
* `profile` - If you specified AAC for Audio:Codec, choose the AAC profile for the output file.
* `signed` - Whether audio samples are represented with negative and positive numbers (signed) or only positive numbers (unsigned). The supported value is Signed. (PCM Only)

The `thumbnails` object supports the following:

* `aspect_ratio` - The aspect ratio of thumbnails. The following values are valid: auto, 1:1, 4:3, 3:2, 16:9
* `format` - The format of thumbnails, if any. Valid formats are jpg and png.
* `interval` - The approximate number of seconds between thumbnails. The value must be an integer. The actual interval can vary by several seconds from one thumbnail to the next.
* `max_height` - The maximum height of thumbnails, in pixels. If you specify auto, Elastic Transcoder uses 1080 (Full HD) as the default value. If you specify a numeric value, enter an even integer between 32 and 3072, inclusive.
* `max_width` - The maximum width of thumbnails, in pixels. If you specify auto, Elastic Transcoder uses 1920 (Full HD) as the default value. If you specify a numeric value, enter an even integer between 32 and 4096, inclusive.
* `padding_policy` - When you set PaddingPolicy to Pad, Elastic Transcoder might add black bars to the top and bottom and/or left and right sides of thumbnails to make the total size of the thumbnails match the values that you specified for thumbnail MaxWidth and MaxHeight settings.
* `resolution` - The width and height of thumbnail files in pixels, in the format WidthxHeight, where both values are even integers. The values cannot exceed the width and height that you specified in the Video:Resolution object. (To better control resolution and aspect ratio of thumbnails, we recommend that you use the thumbnail values `max_width`, `max_height`, `sizing_policy`, and `padding_policy` instead of `resolution` and `aspect_ratio`. The two groups of settings are mutually exclusive. Do not use them together)
* `sizing_policy` - A value that controls scaling of thumbnails. Valid values are: `Fit`, `Fill`, `Stretch`, `Keep`, `ShrinkToFit`, and `ShrinkToFill`.

The `video` object supports the following:

* `aspect_ratio` - The display aspect ratio of the video in the output file. Valid values are: `auto`, `1:1`, `4:3`, `3:2`, `16:9`. (Note; to better control resolution and aspect ratio of output videos, we recommend that you use the values `max_width`, `max_height`, `sizing_policy`, `padding_policy`, and `display_aspect_ratio` instead of `resolution` and `aspect_ratio`.)
* `bit_rate` - The bit rate of the video stream in the output file, in kilobits/second. You can configure variable bit rate or constant bit rate encoding.
* `codec` - The video codec for the output file. Valid values are `gif`, `H.264`, `mpeg2`, `vp8`, and `vp9`.
* `display_aspect_ratio` - The value that Elastic Transcoder adds to the metadata in the output file. If you set DisplayAspectRatio to auto, Elastic Transcoder chooses an aspect ratio that ensures square pixels. If you specify another option, Elastic Transcoder sets that value in the output file.
* `fixed_gop` - Whether to use a fixed value for Video:FixedGOP. Not applicable for containers of type gif. Valid values are true and false.
* `frame_rate` - The frames per second for the video stream in the output file. The following values are valid: `auto`, `10`, `15`, `23.97`, `24`, `25`, `29.97`, `30`, `50`, `60`.
* `keyframes_max_dist` - The maximum number of frames between key frames. Not applicable for containers of type gif.
* `max_frame_rate` - If you specify auto for FrameRate, Elastic Transcoder uses the frame rate of the input video for the frame rate of the output video, up to the maximum frame rate. If you do not specify a MaxFrameRate, Elastic Transcoder will use a default of 30.
* `max_height` - The maximum height of the output video in pixels. If you specify auto, Elastic Transcoder uses 1080 (Full HD) as the default value. If you specify a numeric value, enter an even integer between 96 and 3072, inclusive.
* `max_width` - The maximum width of the output video in pixels. If you specify auto, Elastic Transcoder uses 1920 (Full HD) as the default value. If you specify a numeric value, enter an even integer between 128 and 4096, inclusive.
* `padding_policy` - When you set PaddingPolicy to Pad, Elastic Transcoder might add black bars to the top and bottom and/or left and right sides of the output video to make the total size of the output video match the values that you specified for `max_width` and `max_height`.
* `resolution` - The width and height of the video in the output file, in pixels. Valid values are `auto` and `widthxheight`. (see note for `aspect_ratio`)
* `sizing_policy` - A value that controls scaling of the output video. Valid values are: `Fit`, `Fill`, `Stretch`, `Keep`, `ShrinkToFit`, `ShrinkToFill`.

The `video_watermarks` object supports the following:

* `horizontal_align` - The horizontal position of the watermark unless you specify a nonzero value for `horzontal_offset`.
* `horizontal_offset` - The amount by which you want the horizontal position of the watermark to be offset from the position specified by `horizontal_align`.
* `id` - A unique identifier for the settings for one watermark. The value of Id can be up to 40 characters long. You can specify settings for up to four watermarks.
* `max_height` - The maximum height of the watermark.
* `max_width` - The maximum width of the watermark.
* `opacity` - A percentage that indicates how much you want a watermark to obscure the video in the location where it appears.
* `sizing_policy` - A value that controls scaling of the watermark. Valid values are: `Fit`, `Stretch`, `ShrinkToFit`
* `target` - A value that determines how Elastic Transcoder interprets values that you specified for `video_watermarks.horizontal_offset`, `video_watermarks.vertical_offset`, `video_watermarks.max_width`, and `video_watermarks.max_height`. Valid values are `Content` and `Frame`.
* `vertical_align` - The vertical position of the watermark unless you specify a nonzero value for `vertical_align`. Valid values are `Top`, `Bottom`, `Center`.
* `vertical_offset` - The amount by which you want the vertical position of the watermark to be offset from the position specified by `vertical_align`

The `video_codec_options` map supports the following:

* `Profile` - The codec profile that you want to use for the output file. (H.264/VP8 Only)
* `Level` - The H.264 level that you want to use for the output file. Elastic Transcoder supports the following levels: `1`, `1b`, `1.1`, `1.2`, `1.3`, `2`, `2.1`, `2.2`, `3`, `3.1`, `3.2`, `4`, `4.1` (H.264 only)
* `MaxReferenceFrames` - The maximum number of previously decoded frames to use as a reference for decoding future frames. Valid values are integers 0 through 16. (H.264 only)
* `MaxBitRate` - The maximum number of kilobits per second in the output video. Specify a value between 16 and 62,500 inclusive, or `auto`. (Optional, H.264/MPEG2/VP8/VP9 only)
* `BufferSize` - The maximum number of kilobits in any x seconds of the output video. This window is commonly 10 seconds, the standard segment duration when you're using ts for the container type of the output video. Specify an integer greater than 0. If you specify MaxBitRate and omit BufferSize, Elastic Transcoder sets BufferSize to 10 times the value of MaxBitRate. (Optional, H.264/MPEG2/VP8/VP9 only)
* `InterlacedMode` -  The interlace mode for the output video. (Optional, H.264/MPEG2 Only)
* `ColorSpaceConversion` - The color space conversion Elastic Transcoder applies to the output video. Valid values are `None`, `Bt709toBt601`, `Bt601toBt709`, and `Auto`. (Optional, H.264/MPEG2 Only)
* `ChromaSubsampling` - The sampling pattern for the chroma (color) channels of the output video. Valid values are `yuv420p` and `yuv422p`.
* `LoopCount` - The number of times you want the output gif to loop (Gif only)
