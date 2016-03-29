/* FreeBSD has support for direct io via the DIRECTIO kernel option
 * since 4.9. The GENERIC kernel does not have it enabled as of 10.1.
 * Check the local /usr/src/sys/amd64/conf/GENERIC file and
 * https://www.freebsd.org/doc/en/books/handbook/kernelconfig-config.html
 */

package util

func IsDirectIOSupported(path string) bool {
	return false
}
