package util

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"net"
	"net/netip"
	"strconv"

	"go4.org/netipx"
)

// ValidateIP Takes the ipExpression as argument and tests whether it is a valid IPv4 addr or CIDR block
// If the input is valid, it will be converted to a Prefix struct and returned.
// If not, a custom error will be returned, along with an empty prefix indicating an invalid input.
func ValidateIP(ipExpression string) (netip.Prefix, error) {
	addr, addrErr := netip.ParseAddr(ipExpression)
	prefix, prefixErr := netip.ParsePrefix(ipExpression)

	if addrErr != nil && prefixErr != nil {
		return netip.Prefix{}, fmt.Errorf("%s is neither a valid IPv4 nor CIDR", ipExpression)
	} else if addrErr != nil && prefixErr == nil { // check if parsable as a valid CIDR
		if prefix.Addr().Is6() {
			return netip.Prefix{}, fmt.Errorf("%s is invalid because IPv6 is not supported", ipExpression)
		}
		return prefix, nil
	} // otherwise parsable as a valid IP

	if addr.Is6() {
		return netip.Prefix{}, fmt.Errorf("%s is invalid because IPv6 is not supported", ipExpression)
	}
	const maxMask = 32
	return netip.PrefixFrom(addr, maxMask), nil
}

func GetFirstLastDecimalAddress(ip string) (firstAddr, lastAddr uint32, err error) {
	prefix, err := netip.ParsePrefix(ip)

	if err != nil {
		firstAddr, err = IP4ToInteger(net.ParseIP(ip))
		lastAddr = firstAddr
	} else {
		netIP := net.IP(prefix.Addr().AsSlice())
		firstAddr, err = IP4ToInteger(netIP)

		if err == nil {
			netIP = net.IP(netipx.PrefixLastIP(prefix).AsSlice())
			lastAddr, err = IP4ToInteger(netIP)
		}
	}

	return firstAddr, lastAddr, err
}

func IP4ToInteger(netIP net.IP) (addr uint32, err error) {
	err = binary.Read(bytes.NewBuffer(netIP.To4()), binary.BigEndian, &addr)
	if err != nil {
		err = fmt.Errorf("failed to convert IP %s to integer: %w", netIP, err)
	}
	return addr, err
}

func ConvertToIP4(ipInt uint32) string {
	// need to do two bit shifting and “0xff” masking
	const mask = 0xff
	b0 := strconv.FormatUint(uint64((ipInt>>24)&mask), 10)
	b1 := strconv.FormatUint(uint64((ipInt>>16)&mask), 10)
	b2 := strconv.FormatUint(uint64((ipInt>>8)&mask), 10)
	b3 := strconv.FormatUint(uint64(ipInt&mask), 10)
	return b0 + "." + b1 + "." + b2 + "." + b3
}
