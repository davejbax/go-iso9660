package iso9660

import (
	"bytes"
	"cmp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"io"
	"os"
	"testing"
)

func TestPrimaryVolumeDescriptor_WriteTo(t *testing.T) {
	// This is the Arch Linux 2025.01.01 x86_64 ISO               ( arch btw )
	pvd := &primaryVolumeDescriptor{
		Header: &volumeDescriptor{
			Kind:                    1,
			StandardIdentifier:      [5]uint8{0x43, 0x44, 0x30, 0x30, 0x31},
			VolumeDescriptorVersion: 1,
		},
		SystemIdentifier:               [32]aCharacter{},
		VolumeIdentifier:               [32]dCharacter{0x41, 0x52, 0x43, 0x48, 0x5F, 0x32, 0x30, 0x32, 0x35, 0x30, 0x31},
		VolumeSpaceSize:                uint32BothByte(601104),
		VolumeSetSize:                  uint16BothByte(1),
		VolumeSequenceNumber:           uint16BothByte(1),
		LogicalBlockSize:               uint16BothByte(2048),
		PathTableSize:                  uint32BothByte(186),
		LocationTypeLPathTable:         54,
		LocationTypeLOptionalPathTable: 0,
		LocationTypeMPathTable:         55,
		LocationTypeMOptionalPathTable: 0,
		RootDirectoryRecord: &directoryRecord{
			Length:                        34,
			ExtendedAttributeRecordLength: 0,
			ExtentLocation:                uint32BothByte(35),
			DataLength:                    uint32BothByte(2048),
			RecordingDateAndTime: dateTime{
				YearsSince1900:            125,
				Month:                     1,
				Day:                       1,
				Hour:                      8,
				Minute:                    45,
				Second:                    59,
				GMTOffsetIn15MinIntervals: 0,
			},
			FileFlags:              fileFlagDirectory,
			FileUnitSize:           0,
			InterleaveGapSize:      0,
			VolumeSequenceNumber:   uint16BothByte(1),
			LengthOfFileIdentifier: 1,
			FileIdentifier:         fileIdentifierSelf,
		},
		VolumeSetIdentifier:         [128]dCharacter{},
		PublisherIdentifier:         [128]aCharacter{0x41, 0x52, 0x43, 0x48, 0x20, 0x4C, 0x49, 0x4E, 0x55, 0x58, 0x20, 0x3C, 0x48, 0x54, 0x54, 0x50, 0x53, 0x3A, 0x2F, 0x2F, 0x41, 0x52, 0x43, 0x48, 0x4C, 0x49, 0x4E, 0x55, 0x58, 0x2E, 0x4F, 0x52, 0x47, 0x3E},
		DataPreparerIdentifier:      [128]dCharacter{0x50, 0x52, 0x45, 0x50, 0x41, 0x52, 0x45, 0x44, 0x20, 0x42, 0x59, 0x20, 0x4D, 0x4B, 0x41, 0x52, 0x43, 0x48, 0x49, 0x53, 0x4F},
		ApplicationIdentifier:       [128]aCharacter{0x41, 0x52, 0x43, 0x48, 0x20, 0x4C, 0x49, 0x4E, 0x55, 0x58, 0x20, 0x4C, 0x49, 0x56, 0x45, 0x2F, 0x52, 0x45, 0x53, 0x43, 0x55, 0x45, 0x20, 0x44, 0x56, 0x44},
		CopyrightFileIdentifier:     [37]dCharacter{},
		AbstractFileIdentifier:      [37]dCharacter{},
		BibliographicFileIdentifier: [37]dCharacter{},
		VolumeCreationDateTime: longDateTime{
			YearDigits:                [4]uint8{'2', '0', '2', '5'},
			MonthDigits:               [2]uint8{'0', '1'},
			DayDigits:                 [2]uint8{'0', '1'},
			HourDigits:                [2]uint8{'0', '8'},
			MinuteDigits:              [2]uint8{'4', '5'},
			SecondDigits:              [2]uint8{'1', '0'},
			CentisecondsDigits:        [2]uint8{'0', '0'},
			GMTOffsetIn15MinIntervals: 0,
		},
		VolumeModificationDateTime: longDateTime{
			YearDigits:                [4]uint8{'2', '0', '2', '5'},
			MonthDigits:               [2]uint8{'0', '1'},
			DayDigits:                 [2]uint8{'0', '1'},
			HourDigits:                [2]uint8{'0', '8'},
			MinuteDigits:              [2]uint8{'4', '5'},
			SecondDigits:              [2]uint8{'1', '0'},
			CentisecondsDigits:        [2]uint8{'0', '0'},
			GMTOffsetIn15MinIntervals: 0,
		},
		VolumeExpirationDateTime: zeroLongDateTime,
		VolumeEffectiveDateTime:  zeroLongDateTime,
		FileStructureVersion:     1,
		ApplicationUse:           [512]uint8{},
	}

	fillArray(aCharacter(0x20), pvd.SystemIdentifier[:])
	fillArray(dCharacter(0x20), pvd.VolumeIdentifier[:])
	fillArray(dCharacter(0x20), pvd.VolumeSetIdentifier[:])
	fillArray(aCharacter(0x20), pvd.PublisherIdentifier[:])
	fillArray(dCharacter(0x20), pvd.DataPreparerIdentifier[:])
	fillArray(aCharacter(0x20), pvd.ApplicationIdentifier[:])
	fillArray(0x20, pvd.CopyrightFileIdentifier[:])
	fillArray(0x20, pvd.AbstractFileIdentifier[:])
	fillArray(0x20, pvd.BibliographicFileIdentifier[:])
	fillArray(0x20, pvd.ApplicationUse[:])

	buff := bytes.NewBuffer(nil)

	testdataFile, err := os.Open("testdata/volume_test_pvd.dat")
	if err != nil {
		t.Fatalf("failed to load test data: %v", err)
	}
	defer testdataFile.Close()

	expected, err := io.ReadAll(testdataFile)
	if err != nil {
		t.Fatalf("failed to read test data: %v", err)
	}

	written, err := pvd.WriteTo(buff)
	require.NoError(t, err, "Primary volume descriptor should be written without error")
	assert.Equal(t, written, int64(buff.Len()), "Number of written bytes returned should reflect reality")
	assert.Equal(t, logicalSectorSize, buff.Len(), "Primary volume descriptor should write one logical sector of bytes")
	assert.Equal(t, expected, buff.Bytes(), "Primary volume descriptor should encode correctly and match real-world PVD test data")
}

func fillArray[T cmp.Ordered](filler T, array []T) {
	var zeroVal T
	foundStart := false
	for i := 0; i < len(array); i++ {
		if !foundStart && array[i] == zeroVal {
			foundStart = true
		}

		if !foundStart {
			continue
		}

		array[i] = filler
	}
}
