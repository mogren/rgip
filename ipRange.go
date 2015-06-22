package main

import (
	"encoding/csv"
	"errors"
	"github.com/edsrzf/mmap-go"
	"io"
	"io/ioutil"
	"log"
	"os"
	"reflect"
	"sort"
	"strconv"
	"sync"
	"unsafe"
)

type ipRange struct {
	rangeFrom, rangeTo uint32
	data               int
}

type ipRangeList []ipRange

type ipRanges struct {
	ranges ipRangeList
	sync.RWMutex
}

func (r ipRangeList) Len() int           { return len(r) }
func (r ipRangeList) Less(i, j int) bool { return (r)[i].rangeTo < (r)[j].rangeTo }
func (r ipRangeList) Swap(i, j int)      { (r)[i], (r)[j] = (r)[j], (r)[i] }

func (r ipRangeList) lookup(ip32 uint32) interface{} {
	idx := sort.Search(len(r), func(i int) bool { return ip32 <= r[i].rangeTo })

	if idx != -1 && r[idx].rangeFrom <= ip32 && ip32 <= r[idx].rangeTo {
		return r[idx].data
	}

	return nil
}

func (ipr *ipRanges) lookup(ip32 uint32) int {
	ipr.Lock()
	defer ipr.Unlock()
	data := ipr.ranges.lookup(ip32)

	if data == nil {
		return 0
	}

	return data.(int)
}

func reflectByteSlice(rows []ipRange) []byte {
	header := *(*reflect.SliceHeader)(unsafe.Pointer(&rows))

	size := int(unsafe.Sizeof(ipRange{}))
	header.Len *= size
	header.Cap *= size

	data := *(*[]byte)(unsafe.Pointer(&header))
	return data
}

func reflectIpRangeRows(bytes []byte) ([]ipRange, error) {
	header := *(*reflect.SliceHeader)(unsafe.Pointer(&bytes))

	size := int(unsafe.Sizeof(ipRange{}))
	if header.Len%size != 0 {
		return nil, errors.New("the length of the byte array isn't a multiple of the size of an ipRange")
	}

	header.Len /= size
	header.Cap /= size

	data := *(*[]ipRange)(unsafe.Pointer(&header))
	return data, nil
}

func writeMmap(filename string, ranges []ipRange) {
	representation := reflectByteSlice(ranges)
	ioutil.WriteFile(filename, representation, 0644)
}

func mmapIpRanges(filename string) ([]ipRange, error) {
	file, err := os.OpenFile(filename, os.O_RDONLY, 0644)
	if err != nil {
		return nil, err
	}

	mmapFile, err := mmap.Map(file, mmap.RDONLY, 0)
	if err != nil {
		return nil, err
	}

	return reflectIpRangeRows(mmapFile)
}

func loadIpRangesFromCSV(fname string, transform func(string) (int, error)) (ipRangeList, error) {
	f, err := os.Open(fname)
	if err != nil {
		log.Println("can't open file: ", fname, err)
		return nil, err
	}
	defer f.Close()

	svr := csv.NewReader(f)

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
		if len(r) < 3 {
			ipFrom = prevIP + 1
			ipTo = convert.check(r[0], strconv.Atoi)
			data = convert.check(r[1], transform)
			prevIP = ipTo
		} else {
			ipFrom = convert.check(r[0], strconv.Atoi)
			ipTo = convert.check(r[1], strconv.Atoi)
			data = convert.check(r[2], transform)
		}

		if convert.err != nil {
			log.Printf("error parsing %v: %s", r, err)
			return nil, convert.err
		}

		ips = append(ips, ipRange{rangeFrom: uint32(ipFrom), rangeTo: uint32(ipTo), data: data})
	}

	return ips, nil
}

func loadIpRanges(fname string, transform func(string) (int, error)) (ipRangeList, error) {
	ips, err := mmapIpRanges(fname)
	if err == nil {
		return ips, err
	}

	ips, err = loadIpRangesFromCSV(fname, transform)
	if err != nil {
		return nil, err
	}

	writeMmap(fname, ips)
	return mmapIpRanges(fname)
}
