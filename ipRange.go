package main

import (
	"bytes"
	"encoding/binary"
	"encoding/csv"
	"fmt"
	"io"
	"log"
	"os"
	"reflect"
	"sort"
	"strconv"
	"sync"
	"unsafe"
)

var magicBytes = []byte{'r', 'g', 'i', 'p', 'M', 'a', 'p', 0}

const ipRangeSize int = int(unsafe.Sizeof(ipRange{}))

type ipRange struct {
	rangeFrom, rangeTo uint32
	data               int32
}

type ipRangeList []ipRange

type ipRanges struct {
	ranges ipRangeList
	sync.RWMutex
}

func (r ipRangeList) Len() int           { return len(r) }
func (r ipRangeList) Less(i, j int) bool { return (r)[i].rangeTo < (r)[j].rangeTo }
func (r ipRangeList) Swap(i, j int)      { (r)[i], (r)[j] = (r)[j], (r)[i] }

// lookup returns the found value, if any, followed by a bool indicating whether the value was found
func (r ipRangeList) lookup(ip32 uint32) (int32, bool) {
	idx := sort.Search(len(r), func(i int) bool { return ip32 <= r[i].rangeTo })

	if idx != -1 && r[idx].rangeFrom <= ip32 && ip32 <= r[idx].rangeTo {
		return r[idx].data, true
	}

	return 0, false
}

// lookup returns the found value, if any, followed by a bool indicating whether the value was found
func (ipr *ipRanges) lookup(ip32 uint32) (int32, bool) {
	ipr.Lock()
	defer ipr.Unlock()
	return ipr.ranges.lookup(ip32)
}

func reflectByteSlice(rows []ipRange) []byte {
	header := *(*reflect.SliceHeader)(unsafe.Pointer(&rows))

	header.Len *= ipRangeSize
	header.Cap *= ipRangeSize

	data := *(*[]byte)(unsafe.Pointer(&header))
	return data
}

func readMagicBytes(file *os.File, name string) error {
	b := make([]byte, len(magicBytes))
	n, err := file.Read(b)
	if err != nil {
		return fmt.Errorf("can't read file %s %s", name, err)
	}

	if n != len(magicBytes) {
		return fmt.Errorf("file format is incorrect, expected %d bytes in the %s, got %d", len(magicBytes), name, n)
	}

	if !bytes.Equal(b, magicBytes) {
		return fmt.Errorf("file format is incorrect, expected %s '%s', actual '%s'", name, magicBytes, b)
	}

	return nil
}

func loadIpRangesFromBinary(file *os.File) ([]ipRange, error) {
	err := readMagicBytes(file, "header")
	if err != nil {
		return nil, err
	}

	lenranges := make([]byte, 4)
	n, err := file.Read(lenranges)
	if n != len(lenranges) || err != nil {
		return nil, fmt.Errorf("can't read file size field %s", err)
	}

	ranges := make([]ipRange, binary.LittleEndian.Uint32(lenranges))
	b := make([]byte, ipRangeSize)
	for i := range ranges {
		n, err = file.Read(b)
		if n != ipRangeSize || err != nil {
			return nil, fmt.Errorf("expected %d items, got %d", len(ranges), i)
		}

		ranges[i] = ipRange{
			binary.LittleEndian.Uint32(b[0:]),
			binary.LittleEndian.Uint32(b[4:]),
			int32(binary.LittleEndian.Uint32(b[8:])),
		}
	}

	err = readMagicBytes(file, "footer")
	if err != nil {
		return nil, err
	}

	return ranges, nil
}

func writeBinary(file *os.File, ranges []ipRange) error {
	_, err := file.Write(magicBytes)
	if err != nil {
		return err
	}

	lenranges := make([]byte, 4)
	binary.LittleEndian.PutUint32(lenranges, uint32(len(ranges)))
	_, err = file.Write(lenranges)
	if err != nil {
		return err
	}

	_, err = file.Write(reflectByteSlice(ranges))
	if err != nil {
		return err
	}

	_, err = file.Write(magicBytes)
	return err
}

func loadIpRangesFromCSV(file *os.File) (ipRangeList, error) {
	svr := csv.NewReader(file)

	var ips ipRangeList

	prevIP := -1

	for {
		r, err := svr.Read()
		if err == io.EOF {
			break
		}

		if err != nil {
			log.Println("error reading CSV: ", err)
			return nil, err
		}

		var ipFrom, ipTo, data int

		var convert converr
		ipFrom = prevIP + 1
		ipTo = convert.check(r[0], strconv.Atoi)
		data = convert.check(r[1], strconv.Atoi)
		prevIP = ipTo

		if convert.err != nil {
			log.Printf("error parsing %v: %s", r, err)
			return nil, convert.err
		}

		ips = append(ips, ipRange{rangeFrom: uint32(ipFrom), rangeTo: uint32(ipTo), data: int32(data)})
	}

	return ips, nil
}

func loadIpRanges(fname string, isbinary bool) (ipRangeList, error) {
	file, err := os.Open(fname)
	if err != nil {
		log.Println("can't open file: ", fname, err)
		return nil, err
	}

	defer file.Close()
	if isbinary {
		return loadIpRangesFromBinary(file)
	}

	return loadIpRangesFromCSV(file)
}
