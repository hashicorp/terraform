/*
Package dataurl parses Data URL Schemes
according to RFC 2397
(http://tools.ietf.org/html/rfc2397).

Data URLs are small chunks of data commonly used in browsers to display inline data,
typically like small images, or when you use the FileReader API of the browser.

A dataurl looks like:

	data:text/plain;charset=utf-8,A%20brief%20note

Or, with base64 encoding:

	data:image/vnd.microsoft.icon;name=golang%20favicon;base64,AAABAAEAEBAAAAEAIABoBAAAFgAAACgAAAAQAAAAIAAAAAEAIAAAAAAAAAAAAAAAAAAAAAAAAAAA
	AAAAAAD///8AVE44//7hdv/+4Xb//uF2//7hdv/+4Xb//uF2//7hdv/+4Xb//uF2//7hdv/+4Xb/
	/uF2/1ROOP////8A////AFROOP/+4Xb//uF2//7hdv/+4Xb//uF2//7hdv/+4Xb//uF2//7hdv/+
	...
	/6CcjP97c07/e3NO/1dOMf9BOiX/TkUn/2VXLf97c07/e3NO/6CcjP/h4uX/////AP///wD///8A
	////AP///wD///8A////AP///wDq6/H/3N/j/9fZ3f/q6/H/////AP///wD///8A////AP///wD/
	//8AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA
	AAAAAAAAAAAAAA==

Common functions are Decode and DecodeString to obtain a DataURL,
and DataURL.String() and DataURL.WriteTo to generate a Data URL string.

*/
package dataurl
