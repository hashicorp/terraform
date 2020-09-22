package tablestore

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"math"
)

const (
	HEADER = 0x75

	// tag type
	TAG_ROW_PK             = 0x1
	TAG_ROW_DATA           = 0x2
	TAG_CELL               = 0x3
	TAG_CELL_NAME          = 0x4
	TAG_CELL_VALUE         = 0x5
	TAG_CELL_TYPE          = 0x6
	TAG_CELL_TIMESTAMP     = 0x7
	TAG_DELETE_ROW_MARKER  = 0x8
	TAG_ROW_CHECKSUM       = 0x9
	TAG_CELL_CHECKSUM      = 0x0A
	TAG_EXTENSION          = 0x0B
	TAG_SEQ_INFO           = 0x0C
	TAG_SEQ_INFO_EPOCH     = 0x0D
	TAG_SEQ_INFO_TS        = 0x0E
	TAG_SEQ_INFO_ROW_INDEX = 0x0F

	// cell op type
	DELETE_ALL_VERSION = 0x1
	DELETE_ONE_VERSION = 0x3
	INCREMENT = 0x4;

	// variant type
	VT_INTEGER = 0x0
	VT_DOUBLE  = 0x1
	VT_BOOLEAN = 0x2
	VT_STRING  = 0x3

	//public final static byte VT_NULL = 0x6;
	VT_BLOB           = 0x7
	VT_INF_MIN        = 0x9
	VT_INF_MAX        = 0xa
	VT_AUTO_INCREMENT = 0xb

	LITTLE_ENDIAN_32_SIZE = 4
	LITTLE_ENDIAN_64_SIZE = 8
)

const spaceSize = 256

var crc8Table = make([]byte, spaceSize)

func init() {
	for i := 0; i < spaceSize; i++ {
		x := byte(i)
		for j := 8; j > 0; j-- {
			if (x & 0x80) != 0 {
				x = (x << 1) ^ 0x07
			} else {
				x = (x << 1) ^ 0
			}
		}
		crc8Table[i] = x
	}
}

func crc8Byte(crc, in byte) byte {
	return crc8Table[(crc^in)&0xff]
}

func crc8Int32(crc byte, in int32) byte {
	for i := 0; i < 4; i++ {
		crc = crc8Byte(crc, byte((in & 0xff)))
		in >>= 8
	}

	return crc
}

func crc8Int64(crc byte, in int64) byte {
	for i := 0; i < 8; i++ {
		crc = crc8Byte(crc, byte((in & 0xff)))
		in >>= 8
	}

	return crc
}

func crc8Bytes(crc byte, in []byte) byte {
	for i := 0; i < len(in); i++ {
		crc = crc8Byte(crc, in[i])
	}

	return crc
}

func writeRawByte(w io.Writer, value byte) {
	w.Write([]byte{value})
}

/*func writeRawByteInt8(w io.Writer, value int) {
	w.Write([]byte{byte(value)})
}*/

func writeRawLittleEndian32(w io.Writer, value int32) {
	w.Write([]byte{byte((value) & 0xFF)})
	w.Write([]byte{byte((value >> 8) & 0xFF)})
	w.Write([]byte{byte((value >> 16) & 0xFF)})
	w.Write([]byte{byte((value >> 24) & 0xFF)})
}

func writeRawLittleEndian64(w io.Writer, value int64) {
	w.Write([]byte{byte((value) & 0xFF)})
	w.Write([]byte{byte((value >> 8) & 0xFF)})
	w.Write([]byte{byte((value >> 16) & 0xFF)})
	w.Write([]byte{byte((value >> 24) & 0xFF)})
	w.Write([]byte{byte((value >> 32) & 0xFF)})
	w.Write([]byte{byte((value >> 40) & 0xFF)})
	w.Write([]byte{byte((value >> 48) & 0xFF)})
	w.Write([]byte{byte((value >> 56) & 0xFF)})
}

func writeDouble(w io.Writer, value float64) {
	writeRawLittleEndian64(w, int64(math.Float64bits(value)))
}

func writeBoolean(w io.Writer, value bool) {
	if value {
		w.Write([]byte{byte(1)})
	} else {
		w.Write([]byte{byte(0)})
	}
}

func writeBytes(w io.Writer, value []byte) {
	w.Write(value)
}

func writeHeader(w io.Writer) {
	writeRawLittleEndian32(w, HEADER)
}

func writeTag(w io.Writer, tag byte) {
	writeRawByte(w, tag)
}

func writeCellName(w io.Writer, name []byte) {
	writeTag(w, TAG_CELL_NAME)
	writeRawLittleEndian32(w, int32(len(name)))
	writeBytes(w, name)
}

type PlainBufferCell struct {
	cellName         []byte
	cellValue        *ColumnValue
	cellTimestamp    int64
	cellType         byte
	ignoreValue      bool
	hasCellTimestamp bool
	hasCellType      bool
}

func (cell *PlainBufferCell) writeCell(w io.Writer) {
	writeTag(w, TAG_CELL)
	writeCellName(w, cell.cellName)
	if cell.ignoreValue == false {
		cell.cellValue.writeCellValue(w)
	}

	if cell.hasCellType {
		writeTag(w, TAG_CELL_TYPE)
		writeRawByte(w, cell.cellType)
	}

	if cell.hasCellTimestamp {
		writeTag(w, TAG_CELL_TIMESTAMP)
		writeRawLittleEndian64(w, cell.cellTimestamp)
	}

	writeTag(w, TAG_CELL_CHECKSUM)
	writeRawByte(w, cell.getCheckSum(byte(0x0)))
}

func (cell *PlainBufferCell) getCheckSum(crc byte) byte {
	crc = crc8Bytes(crc, cell.cellName)
	if cell.ignoreValue == false {
		crc = cell.cellValue.getCheckSum(crc)
	}

	if cell.hasCellTimestamp {
		crc = crc8Int64(crc, cell.cellTimestamp)
	}
	if cell.hasCellType {
		crc = crc8Byte(crc, cell.cellType)
	}
	return crc
}

type PlainBufferRow struct {
	primaryKey      []*PlainBufferCell
	cells           []*PlainBufferCell
	hasDeleteMarker bool
	extension       *RecordSequenceInfo // optional
}

func (row *PlainBufferRow) writeRow(w io.Writer) {
	/* pk */
	writeTag(w, TAG_ROW_PK)
	for _, pk := range row.primaryKey {
		pk.writeCell(w)
	}

	if len(row.cells) > 0 {
		writeTag(w, TAG_ROW_DATA)
		for _, cell := range row.cells {
			cell.writeCell(w)
		}
	}

	writeTag(w, TAG_ROW_CHECKSUM)
	writeRawByte(w, row.getCheckSum(byte(0x0)))
}

func (row *PlainBufferRow) writeRowWithHeader(w io.Writer) {
	writeHeader(w)
	row.writeRow(w)
}

func (row *PlainBufferRow) getCheckSum(crc byte) byte {
	for _, cell := range row.primaryKey {
		crcCell := cell.getCheckSum(byte(0x0))
		crc = crc8Byte(crc, crcCell)
	}

	for _, cell := range row.cells {
		crcCell := cell.getCheckSum(byte(0x0))
		crc = crc8Byte(crc, crcCell)
	}

	del := byte(0x0)
	if row.hasDeleteMarker {
		del = byte(0x1)
	}

	crc = crc8Byte(crc, del)

	return crc
}

func readRawByte(r *bytes.Reader) byte {
	if r.Len() == 0 {
		panic(errUnexpectIoEnd)
	}

	b, _ := r.ReadByte()

	return b
}

func readTag(r *bytes.Reader) int {
	return int(readRawByte(r))
}

func readRawLittleEndian64(r *bytes.Reader) int64 {
	if r.Len() < 8 {
		panic(errUnexpectIoEnd)
	}

	var v int64
	binary.Read(r, binary.LittleEndian, &v)

	return v
}

func readRawLittleEndian32(r *bytes.Reader) int32 {
	if r.Len() < 4 {
		panic(errUnexpectIoEnd)
	}

	var v int32
	binary.Read(r, binary.LittleEndian, &v)

	return v
}

func readBoolean(r *bytes.Reader) bool {
	return readRawByte(r) != 0
}

func readBytes(r *bytes.Reader, size int32) []byte {
	if int32(r.Len()) < size {
		panic(errUnexpectIoEnd)
	}
	v := make([]byte, size)
	r.Read(v)
	return v
}

func readCellValue(r *bytes.Reader) *ColumnValue {
	value := new(ColumnValue)
	readRawLittleEndian32(r)
	tp := readRawByte(r)
	switch tp {
	case VT_INTEGER:
		value.Type = ColumnType_INTEGER
		value.Value = readRawLittleEndian64(r)
	case VT_DOUBLE:
		value.Type = ColumnType_DOUBLE
		value.Value = math.Float64frombits(uint64(readRawLittleEndian64(r)))
	case VT_BOOLEAN:
		value.Type = ColumnType_BOOLEAN
		value.Value = readBoolean(r)
	case VT_STRING:
		value.Type = ColumnType_STRING
		value.Value = string(readBytes(r, readRawLittleEndian32(r)))
	case VT_BLOB:
		value.Type = ColumnType_BINARY
		value.Value = []byte(readBytes(r, readRawLittleEndian32(r)))
	}
	return value
}

func readCell(r *bytes.Reader) *PlainBufferCell {
	cell := new(PlainBufferCell)
	tag := readTag(r)
	if tag != TAG_CELL_NAME {
		panic(errTag)
	}

	cell.cellName = readBytes(r, readRawLittleEndian32(r))
	tag = readTag(r)

	if tag == TAG_CELL_VALUE {
		cell.cellValue = readCellValue(r)
		tag = readTag(r)
	}
	if tag == TAG_CELL_TYPE {
		readRawByte(r)
		tag = readTag(r)
	}

	if tag == TAG_CELL_TIMESTAMP {
		cell.cellTimestamp = readRawLittleEndian64(r)
		tag = readTag(r)
	}

	if tag == TAG_CELL_CHECKSUM {
		readRawByte(r)
	} else {
		panic(errNoChecksum)
	}

	return cell
}

func readRowPk(r *bytes.Reader) []*PlainBufferCell {
	primaryKeyColumns := make([]*PlainBufferCell, 0, 4)

	tag := readTag(r)
	for tag == TAG_CELL {
		primaryKeyColumns = append(primaryKeyColumns, readCell(r))
		tag = readTag(r)
	}

	r.Seek(-1, 1)

	return primaryKeyColumns
}

func readRowData(r *bytes.Reader) []*PlainBufferCell {
	columns := make([]*PlainBufferCell, 0, 10)

	tag := readTag(r)
	for tag == TAG_CELL {
		columns = append(columns, readCell(r))
		tag = readTag(r)
	}

	r.Seek(-1, 1)

	return columns
}

func readRow(r *bytes.Reader) *PlainBufferRow {
	row := new(PlainBufferRow)
	tag := readTag(r)
	if tag == TAG_ROW_PK {
		row.primaryKey = readRowPk(r)
		tag = readTag(r)
	}

	if tag == TAG_ROW_DATA {
		row.cells = readRowData(r)
		tag = readTag(r)
	}

	if tag == TAG_DELETE_ROW_MARKER {
		row.hasDeleteMarker = true
		tag = readTag(r)
	}

	if tag == TAG_EXTENSION {
		row.extension = readRowExtension(r)
		tag = readTag(r)
	}

	if tag == TAG_ROW_CHECKSUM {
		readRawByte(r)
	} else {
		panic(errNoChecksum)
	}
	return row
}

func readRowsWithHeader(r *bytes.Reader) (rows []*PlainBufferRow, err error) {
	defer func() {
		if err2 := recover(); err2 != nil {
			if _, ok := err2.(error); ok {
				err = err2.(error)
			}
			return
		}
	}()

	// TODO: panic
	if readRawLittleEndian32(r) != HEADER {
		return nil, fmt.Errorf("Invalid header from plain buffer")
	}

	rows = make([]*PlainBufferRow, 0, 10)

	for r.Len() > 0 {
		rows = append(rows, readRow(r))
	}

	return rows, nil
}

func readRowExtension(r *bytes.Reader) *RecordSequenceInfo {
	readRawLittleEndian32(r) // useless
	tag := readTag(r)
	if tag != TAG_SEQ_INFO {
		panic(errTag)
	}

	readRawLittleEndian32(r) // useless
	tag = readTag(r)
	if tag != TAG_SEQ_INFO_EPOCH {
		panic(errTag)
	}
	epoch := readRawLittleEndian32(r)

	tag = readTag(r)
	if tag != TAG_SEQ_INFO_TS {
		panic(errTag)
	}
	ts := readRawLittleEndian64(r)

	tag = readTag(r)
	if tag != TAG_SEQ_INFO_ROW_INDEX {
		panic(errTag)
	}
	rowIndex := readRawLittleEndian32(r)

	ext := RecordSequenceInfo{}
	ext.Epoch = epoch
	ext.Timestamp = ts
	ext.RowIndex = rowIndex
	return &ext
}
