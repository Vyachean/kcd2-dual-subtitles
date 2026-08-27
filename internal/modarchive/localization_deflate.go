package modarchive

import (
	"archive/zip"
	"bytes"
	"compress/flate"
	"fmt"
	"hash/crc32"
	"math"
)

const kcd2DeflateZIPVersion uint16 = 20

// buildDeflatedLocalizationCryPak emits the ZIP shape used by working retail
// and third-party KCD2 localization PAKs: raw DEFLATE, no data descriptor, no
// extra fields, deterministic DOS timestamps, and Windows ZIP 2.0 version
// fields. Keeping this separate from buildCryPak preserves the already accepted
// stored-entry contract for callers that still require it.
func buildDeflatedLocalizationCryPak(entries []archiveEntry) ([]byte, error) {
	var buffer bytes.Buffer
	writer := zip.NewWriter(&buffer)

	for _, entry := range entries {
		if uint64(len(entry.data)) > math.MaxUint32 {
			_ = writer.Close()
			return nil, fmt.Errorf("CryPak entry %q exceeds 32-bit ZIP size limit", entry.name)
		}

		compressed, err := rawDeflate(entry.data)
		if err != nil {
			_ = writer.Close()
			return nil, fmt.Errorf("compress CryPak entry %q: %w", entry.name, err)
		}
		if uint64(len(compressed)) > math.MaxUint32 {
			_ = writer.Close()
			return nil, fmt.Errorf("compressed CryPak entry %q exceeds 32-bit ZIP size limit", entry.name)
		}

		header := &zip.FileHeader{
			Name:               entry.name,
			Method:             zip.Deflate,
			Flags:              0,
			CreatorVersion:     kcd2DeflateZIPVersion,
			ReaderVersion:      kcd2DeflateZIPVersion,
			CRC32:              crc32.ChecksumIEEE(entry.data),
			CompressedSize:     uint32(len(compressed)),
			UncompressedSize:   uint32(len(entry.data)),
			CompressedSize64:   uint64(len(compressed)),
			UncompressedSize64: uint64(len(entry.data)),
			ModifiedTime:       deterministicDOSTime,
			ModifiedDate:       deterministicDOSDate,
		}

		entryWriter, err := writer.CreateRaw(header)
		if err != nil {
			_ = writer.Close()
			return nil, fmt.Errorf("create compressed CryPak entry %q: %w", entry.name, err)
		}
		if _, err := entryWriter.Write(compressed); err != nil {
			_ = writer.Close()
			return nil, fmt.Errorf("write compressed CryPak entry %q: %w", entry.name, err)
		}
	}

	if err := writer.Close(); err != nil {
		return nil, fmt.Errorf("close compressed CryPak archive: %w", err)
	}
	return buffer.Bytes(), nil
}

func rawDeflate(data []byte) ([]byte, error) {
	var compressed bytes.Buffer
	writer, err := flate.NewWriter(&compressed, flate.BestCompression)
	if err != nil {
		return nil, err
	}
	if _, err := writer.Write(data); err != nil {
		_ = writer.Close()
		return nil, err
	}
	if err := writer.Close(); err != nil {
		return nil, err
	}
	return compressed.Bytes(), nil
}
