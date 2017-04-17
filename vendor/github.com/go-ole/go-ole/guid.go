package ole

var (
	// IID_NULL is null Interface ID, used when no other Interface ID is known.
	IID_NULL = &GUID{0x00000000, 0x0000, 0x0000, [8]byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}}

	// IID_IUnknown is for IUnknown interfaces.
	IID_IUnknown = &GUID{0x00000000, 0x0000, 0x0000, [8]byte{0xC0, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x46}}

	// IID_IDispatch is for IDispatch interfaces.
	IID_IDispatch = &GUID{0x00020400, 0x0000, 0x0000, [8]byte{0xC0, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x46}}

	// IID_IEnumVariant is for IEnumVariant interfaces
	IID_IEnumVariant = &GUID{0x00020404, 0x0000, 0x0000, [8]byte{0xC0, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x46}}

	// IID_IConnectionPointContainer is for IConnectionPointContainer interfaces.
	IID_IConnectionPointContainer = &GUID{0xB196B284, 0xBAB4, 0x101A, [8]byte{0xB6, 0x9C, 0x00, 0xAA, 0x00, 0x34, 0x1D, 0x07}}

	// IID_IConnectionPoint is for IConnectionPoint interfaces.
	IID_IConnectionPoint = &GUID{0xB196B286, 0xBAB4, 0x101A, [8]byte{0xB6, 0x9C, 0x00, 0xAA, 0x00, 0x34, 0x1D, 0x07}}

	// IID_IInspectable is for IInspectable interfaces.
	IID_IInspectable = &GUID{0xaf86e2e0, 0xb12d, 0x4c6a, [8]byte{0x9c, 0x5a, 0xd7, 0xaa, 0x65, 0x10, 0x1e, 0x90}}

	// IID_IProvideClassInfo is for IProvideClassInfo interfaces.
	IID_IProvideClassInfo = &GUID{0xb196b283, 0xbab4, 0x101a, [8]byte{0xB6, 0x9C, 0x00, 0xAA, 0x00, 0x34, 0x1D, 0x07}}
)

// These are for testing and not part of any library.
var (
	// IID_ICOMTestString is for ICOMTestString interfaces.
	//
	// {E0133EB4-C36F-469A-9D3D-C66B84BE19ED}
	IID_ICOMTestString = &GUID{0xe0133eb4, 0xc36f, 0x469a, [8]byte{0x9d, 0x3d, 0xc6, 0x6b, 0x84, 0xbe, 0x19, 0xed}}

	// IID_ICOMTestInt8 is for ICOMTestInt8 interfaces.
	//
	// {BEB06610-EB84-4155-AF58-E2BFF53608B4}
	IID_ICOMTestInt8 = &GUID{0xbeb06610, 0xeb84, 0x4155, [8]byte{0xaf, 0x58, 0xe2, 0xbf, 0xf5, 0x36, 0x80, 0xb4}}

	// IID_ICOMTestInt16 is for ICOMTestInt16 interfaces.
	//
	// {DAA3F9FA-761E-4976-A860-8364CE55F6FC}
	IID_ICOMTestInt16 = &GUID{0xdaa3f9fa, 0x761e, 0x4976, [8]byte{0xa8, 0x60, 0x83, 0x64, 0xce, 0x55, 0xf6, 0xfc}}

	// IID_ICOMTestInt32 is for ICOMTestInt32 interfaces.
	//
	// {E3DEDEE7-38A2-4540-91D1-2EEF1D8891B0}
	IID_ICOMTestInt32 = &GUID{0xe3dedee7, 0x38a2, 0x4540, [8]byte{0x91, 0xd1, 0x2e, 0xef, 0x1d, 0x88, 0x91, 0xb0}}

	// IID_ICOMTestInt64 is for ICOMTestInt64 interfaces.
	//
	// {8D437CBC-B3ED-485C-BC32-C336432A1623}
	IID_ICOMTestInt64 = &GUID{0x8d437cbc, 0xb3ed, 0x485c, [8]byte{0xbc, 0x32, 0xc3, 0x36, 0x43, 0x2a, 0x16, 0x23}}

	// IID_ICOMTestFloat is for ICOMTestFloat interfaces.
	//
	// {BF1ED004-EA02-456A-AA55-2AC8AC6B054C}
	IID_ICOMTestFloat = &GUID{0xbf1ed004, 0xea02, 0x456a, [8]byte{0xaa, 0x55, 0x2a, 0xc8, 0xac, 0x6b, 0x5, 0x4c}}

	// IID_ICOMTestDouble is for ICOMTestDouble interfaces.
	//
	// {BF908A81-8687-4E93-999F-D86FAB284BA0}
	IID_ICOMTestDouble = &GUID{0xbf908a81, 0x8687, 0x4e93, [8]byte{0x99, 0x9f, 0xd8, 0x6f, 0xab, 0x28, 0x4b, 0xa0}}

	// IID_ICOMTestBoolean is for ICOMTestBoolean interfaces.
	//
	// {D530E7A6-4EE8-40D1-8931-3D63B8605001}
	IID_ICOMTestBoolean = &GUID{0xd530e7a6, 0x4ee8, 0x40d1, [8]byte{0x89, 0x31, 0x3d, 0x63, 0xb8, 0x60, 0x50, 0x10}}

	// IID_ICOMEchoTestObject is for ICOMEchoTestObject interfaces.
	//
	// {6485B1EF-D780-4834-A4FE-1EBB51746CA3}
	IID_ICOMEchoTestObject = &GUID{0x6485b1ef, 0xd780, 0x4834, [8]byte{0xa4, 0xfe, 0x1e, 0xbb, 0x51, 0x74, 0x6c, 0xa3}}

	// IID_ICOMTestTypes is for ICOMTestTypes interfaces.
	//
	// {CCA8D7AE-91C0-4277-A8B3-FF4EDF28D3C0}
	IID_ICOMTestTypes = &GUID{0xcca8d7ae, 0x91c0, 0x4277, [8]byte{0xa8, 0xb3, 0xff, 0x4e, 0xdf, 0x28, 0xd3, 0xc0}}

	// CLSID_COMEchoTestObject is for COMEchoTestObject class.
	//
	// {3C24506A-AE9E-4D50-9157-EF317281F1B0}
	CLSID_COMEchoTestObject = &GUID{0x3c24506a, 0xae9e, 0x4d50, [8]byte{0x91, 0x57, 0xef, 0x31, 0x72, 0x81, 0xf1, 0xb0}}

	// CLSID_COMTestScalarClass is for COMTestScalarClass class.
	//
	// {865B85C5-0334-4AC6-9EF6-AACEC8FC5E86}
	CLSID_COMTestScalarClass = &GUID{0x865b85c5, 0x0334, 0x4ac6, [8]byte{0x9e, 0xf6, 0xaa, 0xce, 0xc8, 0xfc, 0x5e, 0x86}}
)

// GUID is Windows API specific GUID type.
//
// This exists to match Windows GUID type for direct passing for COM.
// Format is in xxxxxxxx-xxxx-xxxx-xxxxxxxxxxxxxxxx.
type GUID struct {
	Data1 uint32
	Data2 uint16
	Data3 uint16
	Data4 [8]byte
}

// IsEqualGUID compares two GUID.
//
// Not constant time comparison.
func IsEqualGUID(guid1 *GUID, guid2 *GUID) bool {
	return guid1.Data1 == guid2.Data1 &&
		guid1.Data2 == guid2.Data2 &&
		guid1.Data3 == guid2.Data3 &&
		guid1.Data4[0] == guid2.Data4[0] &&
		guid1.Data4[1] == guid2.Data4[1] &&
		guid1.Data4[2] == guid2.Data4[2] &&
		guid1.Data4[3] == guid2.Data4[3] &&
		guid1.Data4[4] == guid2.Data4[4] &&
		guid1.Data4[5] == guid2.Data4[5] &&
		guid1.Data4[6] == guid2.Data4[6] &&
		guid1.Data4[7] == guid2.Data4[7]
}
