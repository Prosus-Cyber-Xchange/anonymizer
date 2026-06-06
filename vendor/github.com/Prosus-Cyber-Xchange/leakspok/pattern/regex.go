package pattern

import (
	"bytes"
	"context"
	"fmt"
	"net"
	"regexp"
)

//nolint:gosec // just regex patterns
const (
	phonePattern          = `(?:(?:\+?\d{1,3}[-.\s*]?)?(?:\(?\d{3}\)?[-.\s*]?)?\d{3}[-.\s*]?\d{4,6})|(?:(?:(?:\(\+?\d{2}\))|(?:\+?\d{2}))\s*\d{2}\s*\d{3}\s*\d{4})`
	phonesWithExtsPattern = `(?i)(?:(?:\+?1\s*(?:[.-]\s*)?)?(?:\(\s*(?:[2-9]1[02-9]|[2-9][02-8]1|[2-9][02-8][02-9])\s*\)|(?:[2-9]1[02-9]|[2-9][02-8]1|[2-9][02-8][02-9]))\s*(?:[.-]\s*)?)?(?:[2-9]1[02-9]|[2-9][02-9]1|[2-9][02-9]{2})\s*(?:[.-]\s*)?(?:[0-9]{4})(?:\s*(?:#|x\.?|ext\.?|extension)\s*(?:\d+)?)`
	cnpjAlphanumPattern   = `([A-Z0-9]{2}\.?[A-Z0-9]{3}\.?[A-Z0-9]{3}/?[A-Z0-9]{4}-?[A-Z0-9]{2}|[A-Z0-9]{14})`
	linkPattern           = `(?:(?:https?:\/\/)?(?:[a-z0-9.\-]+|www|[a-z0-9.\-])[.](?:[^\s()<>]+|\((?:[^\s()<>]+|(?:\([^\s()<>]+\)))*\))+(?:\((?:[^\s()<>]+|(?:\([^\s()<>]+\)))*\)|[^\s!()\[\]{};:\'".,<>?]))`
	emailPattern          = `(?i)([A-Za-z0-9!#$%&'*+\/=?^_{|.}~-]+@(?:[a-z0-9](?:[a-z0-9-]*[a-z0-9])?\.)+[a-z0-9](?:[a-z0-9-]*[a-z0-9])?)`

	//Ref: https://ihateregex.io/expr/ip/
	ipv4Pattern = `(\b25[0-5]|\b2[0-4][0-9]|\b[01]?[0-9][0-9]?)(\.(25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)){3}`
	//Ref: https://ihateregex.io/expr/ipv6/
	ipv6Pattern = `(([0-9a-fA-F]{1,4}:){7,7}[0-9a-fA-F]{1,4}|([0-9a-fA-F]{1,4}:){1,7}:|([0-9a-fA-F]{1,4}:){1,6}:[0-9a-fA-F]{1,4}|([0-9a-fA-F]{1,4}:){1,5}(:[0-9a-fA-F]{1,4}){1,2}|([0-9a-fA-F]{1,4}:){1,4}(:[0-9a-fA-F]{1,4}){1,3}|([0-9a-fA-F]{1,4}:){1,3}(:[0-9a-fA-F]{1,4}){1,4}|([0-9a-fA-F]{1,4}:){1,2}(:[0-9a-fA-F]{1,4}){1,5}|[0-9a-fA-F]{1,4}:((:[0-9a-fA-F]{1,4}){1,6})|:((:[0-9a-fA-F]{1,4}){1,7}|:)|fe80:(:[0-9a-fA-F]{0,4}){0,4}%[0-9a-zA-Z]{1,}|::(ffff(:0{1,4}){0,1}:){0,1}((25[0-5]|(2[0-4]|1{0,1}[0-9]){0,1}[0-9])\.){3,3}(25[0-5]|(2[0-4]|1{0,1}[0-9]){0,1}[0-9])|([0-9a-fA-F]{1,4}:){1,4}:((25[0-5]|(2[0-4]|1{0,1}[0-9]){0,1}[0-9])\.){3,3}(25[0-5]|(2[0-4]|1{0,1}[0-9]){0,1}[0-9]))`

	creditCardPattern      = `(?:(?:(?:\d{4}[- ]?){3}\d{4}|\d{15,16}))`
	creditCardBasePattern  = `(?:\d[ -]*?){13,16}`
	creditCardAltPattern   = `(?:4[0-9]{12}(?:[0-9]{3})?|[25][1-7][0-9]{14}|6(?:011|5[0-9][0-9])[0-9]{12}|3[47][0-9]{13}|3(?:0[0-5]|[68][0-9])[0-9]{11}|(?:2131|1800|35\d{3})\d{11})`
	streetAddressPattern   = `(?i)\d{1,4} [\w ]{1,20}(?:street|st|avenue|ave|road|rd|highway|hwy|square|sq|trail|trl|drive|dr|court|ct|park|parkway|pkwy|circle|cir|boulevard|blvd)\W?`
	zipCodePattern         = `\b\d{5}(?:[- ]\d{4})?\b`
	poBoxPattern           = `(?i)P\.? ?O\.? Box \d+`
	ssnPattern             = `\b\d{3}[- ]\d{2}[- ]\d{4}`
	guidPattern            = `[0-9a-fA-F]{8}-?[a-fA-F0-9]{4}-?[a-fA-F0-9]{4}-?[a-fA-F0-9]{4}-?[a-fA-F0-9]{12}`
	ibanPattern            = `(?i)[a-zA-Z]{2}[0-9]{2}[\t\f ]?[a-zA-Z0-9]{4}[\t\f ]?[0-9]{4}[\t\f ]?[0-9]{3}([a-zA-Z0-9][\t\f ]?[a-zA-Z0-9]{0,4}[\t\f ]?[a-zA-Z0-9]{0,4}[\t\f ]?[a-zA-Z0-9]{0,4}[\t\f ]?[a-zA-Z0-9]{0,3})?`
	uuid3Pattern           = `(?i)[0-9a-f]{8}-[0-9a-f]{4}-3[0-9a-f]{3}-[0-9a-f]{4}-[0-9a-f]{12}`
	uuid4Pattern           = `(?i)[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}`
	uuid5Pattern           = `(?i)[0-9a-f]{8}-[0-9a-f]{4}-5[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}`
	dnsPattern             = `([a-zA-Z0-9_]{1}[a-zA-Z0-9_-]{0,62}){1}(\.[a-zA-Z0-9_]{1}[a-zA-Z0-9_-]{0,62})*[\._]?`
	urlSchemaPattern       = `(?i)((ftp|tcp|udp|wss?|https?):\/\/)`
	urlUsernamePattern     = `(\S+(:\S*)?@)`
	urlPathPattern         = `((\/|\?|#)[^\s]*)`
	urlPortPattern         = `(:(\d{1,5}))`
	beginWithZeroPattern   = `^0{2,}`
	testNumAPattern        = `^123`
	containsLettersPattern = `[a-zA-Z]`
	containsSpacesPattern  = `\s`
	altIPPattern           = `(([0-9a-fA-F]{1,4}:){7,7}[0-9a-fA-F]{1,4}|([0-9a-fA-F]{1,4}:){1,7}:|([0-9a-fA-F]{1,4}:){1,6}:[0-9a-fA-F]{1,4}|([0-9a-fA-F]{1,4}:){1,5}(:[0-9a-fA-F]{1,4}){1,2}|([0-9a-fA-F]{1,4}:){1,4}(:[0-9a-fA-F]{1,4}){1,3}|([0-9a-fA-F]{1,4}:){1,3}(:[0-9a-fA-F]{1,4}){1,4}|([0-9a-fA-F]{1,4}:){1,2}(:[0-9a-fA-F]{1,4}){1,5}|[0-9a-fA-F]{1,4}:((:[0-9a-fA-F]{1,4}){1,6})|:((:[0-9a-fA-F]{1,4}){1,7}|:)|fe80:(:[0-9a-fA-F]{0,4}){0,4}%[0-9a-zA-Z]{1,}|::(ffff(:0{1,4}){0,1}:){0,1}((25[0-5]|(2[0-4]|1{0,1}[0-9]){0,1}[0-9])\.){3,3}(25[0-5]|(2[0-4]|1{0,1}[0-9]){0,1}[0-9])|([0-9a-fA-F]{1,4}:){1,4}:((25[0-5]|(2[0-4]|1{0,1}[0-9]){0,1}[0-9])\.){3,3}(25[0-5]|(2[0-4]|1{0,1}[0-9]){0,1}[0-9]))`
	urlIPPattern           = `([1-9]\d?|1\d\d|2[01]\d|22[0-3])(\.(1?\d{1,2}|2[0-4]\d|25[0-5])){2}(?:\.([0-9]\d?|1\d\d|2[0-4]\d|25[0-4]))`
	urlSubdomainPattern    = `((www\.)|([a-zA-Z0-9]([-\.][-\._a-zA-Z0-9]+)*))`

	//Ref: https://stackoverflow.com/a/3809435
	urlPattern = `(http(s)?:\/\/.)?(www\.)?[-a-zA-Z0-9@:%._\+~#=]{2,256}\.[a-z]{2,6}\b([-a-zA-Z0-9@:%_\+.~#?&//=]*)`
)

// Compiled regular expressions
var (
	phoneRegexp          = regexp.MustCompile(phonePattern)
	cnpjAlphanumRegexp   = regexp.MustCompile(cnpjAlphanumPattern)
	phonesWithExtsRegexp = regexp.MustCompile(phonesWithExtsPattern)
	emailRegexp          = regexp.MustCompile(emailPattern)
	ipv4Regexp           = regexp.MustCompile(ipv4Pattern)
	ipv6Regexp           = regexp.MustCompile(ipv6Pattern)
	streetAddressRegexp  = regexp.MustCompile(streetAddressPattern)
	zipCodeRegexp        = regexp.MustCompile(zipCodePattern)
	poBoxRegexp          = regexp.MustCompile(poBoxPattern)
	ssnRegexp            = regexp.MustCompile(ssnPattern)
	guidRegexp           = regexp.MustCompile(guidPattern)
	ibanRegexp           = regexp.MustCompile(ibanPattern)
	uuid3Regexp          = regexp.MustCompile(uuid3Pattern)
	uuid4Regexp          = regexp.MustCompile(uuid4Pattern)
	uuid5Regexp          = regexp.MustCompile(uuid5Pattern)
	urlRegexp            = regexp.MustCompile(urlPattern)
)

// Test credit card numbers map for quick lookup
//
//nolint:gochecknoglobals // test card lookup table, initialized once
var testCreditCardNumbers = [][]byte{
	[]byte("4242424242424242"),
	[]byte("4012888888881881"),
	[]byte("4000056655665556"),
	[]byte("5555555555554444"),
	[]byte("5200828282828210"),
	[]byte("5105105105105100"),
	[]byte("378282246310005"),
	[]byte("371449635398431"),
	[]byte("6011111111111117"),
	[]byte("6011000990139424"),
	[]byte("30569309025904"),
	[]byte("38520000023237"),
	[]byte("3530111333300000"),
	[]byte("3566002020360505"),
}

// MatchTestCreditCard returns true if the input matches a known test credit card number.
func MatchTestCreditCard(_ context.Context, s []byte) bool {
	for _, cc := range testCreditCardNumbers {
		if bytes.Equal(cc, s) {
			return true
		}
	}

	return false
}

// MatchRepeatingNumber checks for 5 or more consecutive identical digits
// Although it uses two loops, it is O(n), as each character is processed at most twice, not n times.
func MatchRepeatingNumber(_ context.Context, s []byte) bool {
	// Look for 5 or more consecutive identical digits
	for i := 0; i < len(s); i++ {
		b := s[i]
		// Check if current byte is a digit
		if b < '0' || b > '9' {
			continue
		}

		// Count consecutive identical digits
		count := 1
		for j := i + 1; j < len(s); j++ {
			if s[j] == b {
				count++
			} else {
				break
			}
		}

		// Found 5 or more consecutive identical digits
		if count >= 5 {
			return true
		}
	}

	return false
}

//nolint:gochecknoglobals // Large set of valid file extensions for filename matching
var validFileExtensions = map[string]struct{}{
	"ez": {}, "anx": {}, "atom": {}, "webp": {}, "atomcat": {}, "atomsrv": {}, "lin": {}, "cu": {}, "davmount": {}, "dcm": {}, "tsp": {}, "es": {}, "otf": {}, "ttf": {}, "pfr": {}, "woff": {}, "spl": {}, "gz": {}, "hta": {}, "jar": {}, "ser": {}, "class": {}, "js": {}, "json": {}, "m3g": {}, "hqx": {}, "cpt": {}, "nb": {}, "nbp": {}, "mbox": {}, "mdb": {}, "doc": {}, "dot": {}, "mxf": {}, "bin": {}, "deploy": {}, "msu": {}, "msp": {}, "oda": {}, "opf": {}, "ogx": {}, "one": {}, "onetoc2": {}, "onetmp": {}, "onepkg": {}, "pdf": {}, "pgp": {}, "key": {}, "sig": {}, "prf": {}, "ps": {}, "ai": {}, "eps": {}, "epsi": {}, "epsf": {}, "eps2": {}, "eps3": {}, "rar": {}, "rdf": {}, "rtf": {}, "stl": {}, "smi": {}, "smil": {}, "xhtml": {}, "xht": {}, "xml": {}, "xsd": {}, "xsl": {}, "xslt": {}, "xspf": {}, "zip": {}, "apk": {}, "cdy": {}, "deb": {}, "ddeb": {}, "udeb": {}, "sfd": {}, "kml": {}, "kmz": {}, "xul": {}, "xls": {}, "xlb": {}, "xlt": {}, "xlam": {}, "xlsb": {}, "xlsm": {}, "xltm": {}, "eot": {}, "thmx": {}, "cat": {}, "ppt": {}, "pps": {}, "ppam": {}, "pptm": {}, "sldm": {}, "ppsm": {}, "potm": {}, "docm": {}, "dotm": {}, "odc": {}, "odb": {}, "odf": {}, "odg": {}, "otg": {}, "odi": {}, "odp": {}, "otp": {}, "ods": {}, "ots": {}, "odt": {}, "odm": {}, "ott": {}, "oth": {}, "pptx": {}, "sldx": {}, "ppsx": {}, "potx": {}, "xlsx": {}, "xltx": {}, "docx": {}, "dotx": {}, "cod": {}, "mmf": {}, "sdc": {}, "sds": {}, "sda": {}, "sdd": {}, "sdf": {}, "sdw": {}, "sgl": {}, "sxc": {}, "stc": {}, "sxd": {}, "std": {}, "sxi": {}, "sti": {}, "sxm": {}, "sxw": {}, "sxg": {}, "stw": {}, "sis": {}, "cap": {}, "pcap": {}, "vsd": {}, "vst": {}, "vsw": {}, "vss": {}, "wbxml": {}, "wmlc": {}, "wmlsc": {}, "wpd": {}, "wp5": {}, "wk": {}, "7z": {}, "abw": {}, "dmg": {}, "bcpio": {}, "torrent": {}, "cab": {}, "cbr": {}, "cbz": {}, "cdf": {}, "cda": {}, "vcd": {}, "pgn": {}, "mph": {}, "cpio": {}, "csh": {}, "dcr": {}, "dir": {}, "dxr": {}, "dms": {}, "wad": {}, "dvi": {}, "pfa": {}, "pfb": {}, "gsf": {}, "pcf": {}, "pcf.Z": {}, "mm": {}, "gan": {}, "gnumeric": {}, "sgf": {}, "gcf": {}, "gtar": {}, "tgz": {}, "taz": {}, "hdf": {}, "hwp": {}, "ica": {}, "info": {}, "ins": {}, "isp": {}, "iii": {}, "iso": {}, "jam": {}, "jnlp": {}, "jmz": {}, "chrt": {}, "kil": {}, "skp": {}, "skd": {}, "skt": {}, "skm": {}, "kpr": {}, "kpt": {}, "ksp": {}, "kwd": {}, "kwt": {}, "latex": {}, "lha": {}, "lyx": {}, "lzh": {}, "lzx": {}, "frm": {}, "maker": {}, "frame": {}, "fm": {}, "fb": {}, "book": {}, "fbdoc": {}, "mif": {}, "m3u8": {}, "application": {}, "manifest": {}, "wmd": {}, "wmz": {}, "com": {}, "exe": {}, "bat": {}, "dll": {}, "msi": {}, "nc": {}, "pac": {}, "nwc": {}, "o": {}, "oza": {}, "p7r": {}, "crl": {}, "pyc": {}, "pyo": {}, "qgs": {}, "shp": {}, "shx": {}, "qtl": {}, "rdp": {}, "rpm": {}, "rss": {}, "rb": {}, "sci": {}, "sce": {}, "xcos": {}, "sh": {}, "shar": {}, "swf": {}, "swfl": {}, "scr": {}, "sql": {}, "sit": {}, "sitx": {}, "sv4cpio": {}, "sv4crc": {}, "tar": {}, "tcl": {}, "gf": {}, "pk": {}, "texinfo": {}, "texi": {}, "~": {}, "%": {}, "bak": {}, "old": {}, "sik": {}, "t": {}, "tr": {}, "roff": {}, "man": {}, "me": {}, "ms": {}, "ustar": {}, "src": {}, "wz": {}, "crt": {}, "xcf": {}, "fig": {}, "xpi": {}, "xz": {}, "amr": {}, "awb": {}, "axa": {}, "au": {}, "snd": {}, "csd": {}, "orc": {}, "sco": {}, "flac": {}, "mid": {}, "midi": {}, "kar": {}, "mpga": {}, "mpega": {}, "mp2": {}, "mp3": {}, "m4a": {}, "m3u": {}, "oga": {}, "ogg": {}, "opus": {}, "spx": {}, "sid": {}, "aif": {}, "aiff": {}, "aifc": {}, "gsm": {}, "wma": {}, "wax": {}, "ra": {}, "rm": {}, "ram": {}, "pls": {}, "sd2": {}, "wav": {}, "alc": {}, "cac": {}, "cache": {}, "csf": {}, "cbin": {}, "cascii": {}, "ctab": {}, "cdx": {}, "cer": {}, "c3d": {}, "chm": {}, "cif": {}, "cmdf": {}, "cml": {}, "cpa": {}, "bsd": {}, "csml": {}, "csm": {}, "ctx": {}, "cxf": {}, "cef": {}, "emb": {}, "embl": {}, "spc": {}, "inp": {}, "gam": {}, "gamin": {}, "fch": {}, "fchk": {}, "cub": {}, "gau": {}, "gjc": {}, "gjf": {}, "gal": {}, "gcg": {}, "gen": {}, "hin": {}, "istr": {}, "ist": {}, "jdx": {}, "dx": {}, "kin": {}, "mcm": {}, "mmd": {}, "mmod": {}, "mol": {}, "rd": {}, "rxn": {}, "sd": {}, "tgf": {}, "mcif": {}, "mol2": {}, "b": {}, "gpt": {}, "mop": {}, "mopcrt": {}, "mpc": {}, "zmt": {}, "moo": {}, "mvb": {}, "asn": {}, "prt": {}, "ent": {}, "val": {}, "aso": {}, "pdb": {}, "ros": {}, "sw": {}, "vms": {}, "vmd": {}, "xtel": {}, "xyz": {}, "gif": {}, "ief": {}, "jp2": {}, "jpg2": {}, "jpeg": {}, "jpg": {}, "jpe": {}, "jpm": {}, "jpx": {}, "jpf": {}, "pcx": {}, "png": {}, "svg": {}, "svgz": {}, "tiff": {}, "tif": {}, "djvu": {}, "djv": {}, "ico": {}, "wbmp": {}, "cr2": {}, "crw": {}, "ras": {}, "cdr": {}, "pat": {}, "cdt": {}, "erf": {}, "art": {}, "jng": {}, "bmp": {}, "nef": {}, "orf": {}, "psd": {}, "pnm": {}, "pbm": {}, "pgm": {}, "ppm": {}, "rgb": {}, "xbm": {}, "xpm": {}, "xwd": {}, "eml": {}, "igs": {}, "iges": {}, "msh": {}, "mesh": {}, "silo": {}, "vrml": {}, "x3dv": {}, "x3d": {}, "x3db": {}, "appcache": {}, "ics": {}, "icz": {}, "css": {}, "csv": {}, "323": {}, "html": {}, "htm": {}, "shtml": {}, "uls": {}, "mml": {}, "asc": {}, "txt": {}, "text": {}, "pot": {}, "brf": {}, "srt": {}, "rtx": {}, "sct": {}, "wsc": {}, "tm": {}, "tsv": {}, "ttl": {}, "vcf": {}, "vcard": {}, "jad": {}, "wml": {}, "wmls": {}, "bib": {}, "boo": {}, "hpp": {}, "hxx": {}, "hh": {}, "cpp": {}, "cxx": {}, "cc": {}, "h": {}, "htc": {}, "c": {}, "d": {}, "diff": {}, "patch": {}, "hs": {}, "java": {}, "ly": {}, "lhs": {}, "moc": {}, "p": {}, "pas": {}, "gcd": {}, "pl": {}, "pm": {}, "py": {}, "scala": {}, "etx": {}, "sfv": {}, "tk": {}, "tex": {}, "ltx": {}, "sty": {}, "cls": {}, "vcs": {}, "3gp": {}, "axv": {}, "dl": {}, "dif": {}, "dv": {}, "fli": {}, "gl": {}, "mpeg": {}, "mpg": {}, "mpe": {}, "ts": {}, "mp4": {}, "qt": {}, "mov": {}, "ogv": {}, "webm": {}, "mxu": {}, "flv": {}, "lsf": {}, "lsx": {}, "mng": {}, "asf": {}, "asx": {}, "wm": {}, "wmv": {}, "wmx": {}, "wvx": {}, "avi": {}, "movie": {}, "mpv": {}, "mkv": {}, "ice": {}, "sisx": {}, "vrm": {}, "wrl": {},
}

// isValidFilenameChar checks if a byte is a valid filename character
// Valid characters: a-zA-Z0-9, hyphen, underscore, dot, plus
func isValidFilenameChar(b byte) bool {
	return (b >= 'a' && b <= 'z') ||
		(b >= 'A' && b <= 'Z') ||
		(b >= '0' && b <= '9') ||
		b == '-' || b == '_' || b == '.' || b == '+'
}

// MatchFilename returns true if the input looks like a filename with a recognized
// extension (e.g., .pdf, .jpg, .go). It uses a large set of known file extensions
// to avoid false positives on random text.
func MatchFilename(_ context.Context, s []byte) bool {
	if len(s) < 3 { // minimum: "a.b"
		return false
	}

	// Find the last dot in the filename
	dotIdx := bytes.LastIndexByte(s, '.')
	if dotIdx <= 0 || dotIdx == len(s)-1 {
		return false // no dot, dot at start, or dot at end
	}

	// Validate that all characters in the filename part are valid
	filename := s[:dotIdx]
	for _, b := range filename {
		if !isValidFilenameChar(b) {
			return false
		}
	}

	// Extract and validate the extension (case-insensitive)
	extension := s[dotIdx+1:]

	// Check if extension is in the valid set
	_, exists := validFileExtensions[string(bytes.ToLower(extension))]
	return exists
}

// MatchPhone returns true if the input matches an international phone number pattern
// using a regular expression.
func MatchPhone(_ context.Context, s []byte) bool {
	return phoneRegexp.Match(s)
}

// MatchPhoneV2 returns true if the input matches a phone number using pure Go
// validation. It supports North American style (e.g., +1-555-123-4567, 5551234567)
// and structured European style (e.g., +33 20 345 6789, (+33) 20 345 6789).
func MatchPhoneV2(_ context.Context, s []byte) bool {
	if len(s) == 0 {
		return false
	}

	// Try Format 1 first
	if matchPhoneFormat1(s) {
		return true
	}

	// Try Format 2
	if matchPhoneFormat2(s) {
		return true
	}

	return false
}

// phoneFormat1Separators contains the separators that should be removed for Format 1
//
//nolint:gochecknoglobals // Separators for phone format 1 validation
var phoneFormat1Separators = map[rune]struct{}{
	' ': {}, '-': {}, '.': {}, '*': {}, '(': {}, ')': {},
}

// matchPhoneFormat1 validates Format 1: Flexible North American Style
// Examples: +1-555-123-4567, (555) 123-4567, 5551234567, +33 20 345 6789
func matchPhoneFormat1(s []byte) bool {
	// Step 1: Sanitization - remove separators: space, hyphen, period, asterisk, parentheses
	sanitized := stripChars(phoneFormat1Separators, s)

	if len(sanitized) == 0 {
		return false
	}

	// Step 2: Plus sign validation
	hasPlusSign := false
	if sanitized[0] == '+' {
		hasPlusSign = true
	} else if bytes.Contains(sanitized, []byte("+")) {
		// Plus sign appears somewhere other than the beginning
		return false
	}

	// Step 3: Digit extraction
	var digitPortion []byte
	if hasPlusSign {
		digitPortion = sanitized[1:]
	} else {
		digitPortion = sanitized
	}

	// Step 4: Verify all characters are digits
	if !isAllDigits(digitPortion) {
		return false
	}

	// Step 5: Length validation
	length := len(digitPortion)
	if hasPlusSign {
		// With country code: 10-13 digits
		return length >= 10 && length <= 13
	}

	// Without country code: exactly 10 digits (domestic)
	return length == 10
}

// matchPhoneFormat2 validates Format 2: Structured European Style
// Examples: +33 20 345 6789, (+33) 20 345 6789
func matchPhoneFormat2(s []byte) bool {
	// Step 1: Whitespace trimming
	trimmed := bytes.TrimSpace(s)
	if len(trimmed) == 0 {
		return false
	}

	// Step 2: Format detection
	if bytes.HasPrefix(trimmed, []byte("(+")) {
		return validatePhoneFormat2Parenthetical(trimmed)
	} else if bytes.HasPrefix(trimmed, []byte("+")) {
		return validatePhoneFormat2Plus(trimmed)
	}

	return false
}

// isAllDigits checks if a byte slice contains only digits
func isAllDigits(b []byte) bool {
	for _, c := range b {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}

// validatePhoneFormat2Parenthetical validates Format 2 with parenthetical country code: (+XX) xx xxx xxxx
func validatePhoneFormat2Parenthetical(s []byte) bool {
	// Step 3a: Find closing parenthesis
	closingParenIdx := bytes.IndexByte(s, ')')
	if closingParenIdx == -1 {
		return false
	}

	// Step 3b: Extract and validate country code (+XX)
	countryCodePart := s[1:closingParenIdx]
	if len(countryCodePart) != 3 || countryCodePart[0] != '+' {
		return false
	}

	if !isAllDigits(countryCodePart[1:]) {
		return false
	}

	// Step 3c: Check for exactly one space after closing parenthesis
	if closingParenIdx+1 >= len(s) || s[closingParenIdx+1] != ' ' {
		return false
	}

	// Step 3d: Split remainder by spaces (should get exactly 3 parts)
	remainder := s[closingParenIdx+2:]
	parts := bytes.Split(remainder, []byte(" "))
	if len(parts) != 3 {
		return false
	}

	// Step 3e: Validate each part - exactly 2, 3, and 4 digits respectively
	return len(parts[0]) == 2 && isAllDigits(parts[0]) &&
		len(parts[1]) == 3 && isAllDigits(parts[1]) &&
		len(parts[2]) == 4 && isAllDigits(parts[2])
}

// validatePhoneFormat2Plus validates Format 2 with plus country code: +XX xx xxx xxxx
func validatePhoneFormat2Plus(s []byte) bool {
	// Step 4a: Split by space (should get exactly 4 parts)
	parts := bytes.Split(s, []byte(" "))
	if len(parts) != 4 {
		return false
	}

	// Step 4b: Validate first part (country code: +XX)
	if len(parts[0]) != 3 || parts[0][0] != '+' {
		return false
	}

	if !isAllDigits(parts[0][1:]) {
		return false
	}

	// Step 4c: Validate remaining three parts - exactly 2, 3, and 4 digits respectively
	return len(parts[1]) == 2 && isAllDigits(parts[1]) &&
		len(parts[2]) == 3 && isAllDigits(parts[2]) &&
		len(parts[3]) == 4 && isAllDigits(parts[3])
}

// MatchPhonesWithExts returns true if the input matches a phone number with
// an extension (e.g., #123, x456, ext.789).
func MatchPhonesWithExts(_ context.Context, s []byte) bool {
	return phonesWithExtsRegexp.Match(s)
}

// MatchEmail returns true if the input matches an email address pattern using
// a regular expression, with an additional check to reject IP-based domains.
func MatchEmail(_ context.Context, s []byte) bool {
	if emailRegexp.Match(s) {
		// Since the golang regex engine does not support lookbehinds, we need to validate
		// cases such as the email address is <name>@1.2.3
		for i := len(s) - 1; i >= 0; i-- {
			if s[i] == '@' {
				break
			}

			if (s[i] < '0' || s[i] > '9') && s[i] != '.' {
				return true
			}
		}
	}

	return false
}

// isDomainAllNumeric checks whether the portion of the input after the '@'
// character consists exclusively of digits and dots. Such patterns indicate
// IP addresses rather than real email domains and should be rejected.
func isDomainAllNumeric(s []byte) bool {
	for i := len(s) - 1; i >= 0; i-- {
		if s[i] == '@' {
			break
		}

		if (s[i] < '0' || s[i] > '9') && s[i] != '.' {
			return false // found a non-digit, non-dot character — real domain
		}
	}

	return true // entire domain portion is digits and dots — likely an IP address
}

// MatchEmailV2 returns true if the input matches a valid email address using
// pure Go validation. It requires exactly one '@', a non-empty local part,
// a domain with a dot, and rejects purely numeric domains (e.g., user@1.2.3).
func MatchEmailV2(_ context.Context, s []byte) bool {
	if len(s) < 3 {
		return false
	}

	at := bytes.IndexByte(s, '@')

	// Need exactly one '@', not first or last.
	if at <= 0 || at != bytes.LastIndexByte(s, '@') || at == len(s)-1 {
		return false
	}

	local, domain := s[:at], s[at+1:]
	// No empty local or domain.
	if len(local) == 0 || len(domain) == 0 {
		return false
	}

	// Domain must contain a dot that is not at edges.
	dot := bytes.LastIndexByte(domain, '.')
	if dot <= 0 || dot == len(domain)-1 {
		return false
	}

	// Reject email-like patterns with purely numeric domains (e.g., <name>@1.2.3)
	if isDomainAllNumeric(s) {
		return false
	}

	return true
}

func stripChars(removalSet map[rune]struct{}, b []byte) []byte {
	return bytes.Map(func(r rune) rune {
		if _, ok := removalSet[r]; ok {
			return -1 // drop this rune
		}
		return r
	}, b)
}

// MatchIP returns true if the input is a valid IPv4 or IPv6 address as determined
// by net.ParseIP, after stripping common wrapping characters (quotes, brackets, etc.).
func MatchIP(_ context.Context, s []byte) bool {
	especialChars := map[rune]struct{}{
		'"': {}, ',': {}, '[': {}, ']': {}, '{': {}, '}': {},
		'-': {}, ';': {}, '?': {}, '!': {}, '`': {}, '\'': {},
	}
	s = stripChars(especialChars, s)

	parsedIP := net.ParseIP(string(s))
	return parsedIP != nil
}

// MatchIPV4 returns true if the input matches an IPv4 address pattern after
// stripping common wrapping characters and validating with net.ParseIP.
func MatchIPV4(_ context.Context, s []byte) bool {
	chars := []string{`"`, `,`, `[`, `]`, `{`, `}`, `-`, `;`, `?`, `!`, "`", "'"}

	for _, char := range chars {
		s = bytes.ReplaceAll(s, []byte(char), []byte(""))
	}

	if ipv4Regexp.Match(s) {
		parsedIP := net.ParseIP(string(s))
		if parsedIP != nil {
			return true
		}
	}

	return false
}

// MatchIPV6 returns true if the input matches an IPv6 address pattern after
// stripping common wrapping characters and validating with net.ParseIP.
func MatchIPV6(_ context.Context, s []byte) bool {
	chars := []string{`"`, `,`, `[`, `]`, `{`, `}`, `-`, `;`, `?`, `!`, "`", "'"}

	for _, char := range chars {
		s = bytes.ReplaceAll(s, []byte(char), []byte(""))
	}

	if ipv6Regexp.Match(s) {
		parsedIP := net.ParseIP(string(s))
		if parsedIP != nil {
			return true
		}
	}

	return false
}

// MatchVisaCreditCard returns true if the input is a valid Visa credit card number:
// exactly 16 digits starting with '4' (after stripping spaces and dashes).
func MatchVisaCreditCard(_ context.Context, input []byte) bool {
	ccStripSet := map[rune]struct{}{
		' ': {},
		'-': {},
	}
	// strip spaces and dashes
	clean := stripChars(ccStripSet, input)

	// must be exactly 16 digits
	if len(clean) != 16 {
		return false
	}

	// must all be digits
	for _, c := range clean {
		if c < '0' || c > '9' {
			return false
		}
	}

	// must start with '4'
	return clean[0] == '4'
}

// MatchMasterCardCreditCard returns true if the input is a valid MasterCard credit
// card number: exactly 16 digits starting with 51 through 55 (after stripping
// spaces and dashes).
func MatchMasterCardCreditCard(_ context.Context, input []byte) bool {
	ccStripSet := map[rune]struct{}{
		' ': {},
		'-': {},
	}
	// strip spaces and dashes
	clean := stripChars(ccStripSet, input)

	// must be exactly 16 digits
	if len(clean) != 16 {
		return false
	}

	// must all be digits
	for _, c := range clean {
		if c < '0' || c > '9' {
			return false
		}
	}

	// must start with 51–55
	if clean[0] != '5' {
		return false
	}
	if clean[1] < '1' || clean[1] > '5' {
		return false
	}

	return true
}

// isValidVINChar checks if a byte is a valid VIN character.
// VINs use uppercase letters A-Z and digits 0-9, excluding I, O, Q per ISO 3779.
func isValidVINChar(b byte) bool {
	switch b {
	case 'I', 'i', 'O', 'o', 'Q', 'q':
		return false
	}
	return (b >= 'A' && b <= 'Z') || (b >= 'a' && b <= 'z') || (b >= '0' && b <= '9')
}

// MatchVIN checks if the input is a valid Vehicle Identification Number.
// VINs are exactly 17 alphanumeric characters excluding I, O, Q per ISO 3779.
func MatchVIN(_ context.Context, s []byte) bool {
	if len(s) != 17 {
		return false
	}
	for _, b := range s {
		if !isValidVINChar(b) {
			return false
		}
	}
	return true
}

// MatchStreetAddress returns true if the input matches a street address pattern
// (e.g., "123 Main Street").
func MatchStreetAddress(_ context.Context, s []byte) bool {
	return streetAddressRegexp.Match(s)
}

// MatchZipCode returns true if the input matches a US ZIP code pattern
// (5 digits optionally followed by a dash and 4 more digits).
func MatchZipCode(_ context.Context, s []byte) bool {
	return zipCodeRegexp.Match(s)
}

// MatchPOBox returns true if the input matches a PO Box address pattern
// (e.g., "P.O. Box 1234").
func MatchPOBox(_ context.Context, s []byte) bool {
	return poBoxRegexp.Match(s)
}

// MatchSSN returns true if the input matches a US Social Security Number pattern
// (9 digits with dashes: XXX-XX-XXXX).
func MatchSSN(_ context.Context, s []byte) bool {
	return ssnRegexp.Match(s)
}

// Matchguid returns true if the input matches a GUID/UUID hex pattern with
// optional dashes between groups.
func Matchguid(_ context.Context, s []byte) bool {
	return guidRegexp.Match(s)
}

// MatchIBAN returns true if the input matches an International Bank Account
// Number (IBAN) pattern.
func MatchIBAN(_ context.Context, s []byte) bool {
	return ibanRegexp.Match(s)
}

func isHexDigit(b byte) bool {
	// accept 0-9, a-f, A-F
	if b >= '0' && b <= '9' {
		return true
	}
	if b >= 'a' && b <= 'f' {
		return true
	}
	if b >= 'A' && b <= 'F' {
		return true
	}
	return false
}

// MatchUUID returns true if the input is exactly 36 characters matching the
// 8-4-4-4-12 UUID format with hex digits and mandatory dashes.
func MatchUUID(_ context.Context, s []byte) bool {
	if len(s) != 36 {
		return false
	}

	for i := 0; i < 36; i++ {
		c := s[i]

		switch i {
		// dash positions: 8-4-4-4-12 => indices 8, 13, 18, 23
		case 8, 13, 18, 23:
			if c != '-' {
				return false
			}
		default:
			if !isHexDigit(c) {
				return false
			}
		}
	}

	return true
}

// MatchUUIDV3 returns true if the input matches a version 3 UUID pattern.
func MatchUUIDV3(_ context.Context, s []byte) bool {
	return uuid3Regexp.Match(s)
}

// MatchUUIDV4 returns true if the input matches a version 4 UUID pattern.
func MatchUUIDV4(_ context.Context, s []byte) bool {
	return uuid4Regexp.Match(s)
}

// MatchUUIDV5 returns true if the input matches a version 5 UUID pattern.
func MatchUUIDV5(_ context.Context, s []byte) bool {
	return uuid5Regexp.Match(s)
}

// MatchURL returns true if the input matches a URL pattern (e.g., https://example.com/path).
func MatchURL(_ context.Context, s []byte) bool {
	match := urlRegexp.Match(s)

	return match
}

// validateCPFCheckDigits calculates and validates the two check digits of a CPF.
// cpf must be exactly 11 ASCII digit bytes.
func validateCPFCheckDigits(cpf []byte) bool {
	// Calculate first check digit
	var sum int
	for i := 0; i < 9; i++ {
		digit, _ := byteAtoi(cpf[i])
		sum += digit * (10 - i)
	}

	result1 := (sum * 10) % 11
	if result1 == 10 {
		result1 = 0
	}

	// Calculate second check digit
	sum = 0
	for i := 0; i < 9; i++ {
		digit, _ := byteAtoi(cpf[i])
		sum += digit * (11 - i)
	}
	sum += result1 * 2
	result2 := (sum * 10) % 11
	if result2 == 10 {
		result2 = 0
	}

	checkDigit1, _ := byteAtoi(cpf[9])
	checkDigit2, _ := byteAtoi(cpf[10])
	return checkDigit1 == result1 && checkDigit2 == result2
}

// MatchCPF returns a Brazilian CPF match
func MatchCPF(_ context.Context, s []byte) bool {
	// Extract only digits from the input
	cpf := bytes.Map(func(r rune) rune {
		if r < '0' || r > '9' {
			return -1 // drop this rune
		}
		return r
	}, s)

	// A valid CPF must have exactly 11 digits
	if len(cpf) != 11 {
		return false
	}

	return validateCPFCheckDigits(cpf)
}

// CNPJ validation constants
//
//nolint:gochecknoglobals // Tables for CNPJ validation
var (
	cnpjFirstDigitTable  = []int{5, 4, 3, 2, 9, 8, 7, 6, 5, 4, 3, 2}
	cnpjSecondDigitTable = []int{6, 5, 4, 3, 2, 9, 8, 7, 6, 5, 4, 3, 2}
)

// MatchCNPJ returns a Brazilian CNPJ match (legacy numeric format only)
func MatchCNPJ(_ context.Context, s []byte) bool {
	cnpj := bytes.Map(func(r rune) rune {
		if r < '0' || r > '9' {
			return -1 // drop this rune
		}
		return r
	}, s)

	// A valid CNPJ must have 14 digits without punctuations
	if len(cnpj) != 14 {
		return false
	}

	numericCPNJ := make([]int, len(cnpj))
	for i, r := range cnpj {
		digit, err := byteAtoi(r)
		if err != nil {
			return false
		}
		numericCPNJ[i] = digit
	}

	d1 := checksum(numericCPNJ, cnpjFirstDigitTable)
	d2 := checksum(numericCPNJ, cnpjSecondDigitTable)

	return numericCPNJ[12] == d1 && numericCPNJ[13] == d2
}

// cleanCNPJFormat removes formatting characters (dots, slashes, hyphens, quotes, brackets)
// from a CNPJ input, leaving only alphanumeric characters.
func cleanCNPJFormat(cnpj []byte) []byte {
	return bytes.Join(bytes.FieldsFunc(cnpj, func(r rune) bool {
		return r == '.' || r == '/' || r == '-' || r == '"' || r == ']' || r == '}'
	}), []byte(""))
}

// cnpjAlphaToNumeric converts uppercase alphanumeric CNPJ bytes to numeric values.
// Digits 0-9 map to 0-9; letters A-Z map to values 10-35 via ASCII offset.
func cnpjAlphaToNumeric(cnpjUpper []byte) []int {
	numericValues := make([]int, 14)
	for i, b := range cnpjUpper {
		numericValues[i] = int(b - 48) // ASCII '0' offset
	}
	return numericValues
}

// MatchCNPJV2 returns a Brazilian CNPJ match for the new alphanumeric format only
// The new format consists of 14 alphanumeric characters (letters A-Z and digits 0-9) and must contain at least one letter
// Reference: https://www.gov.br/receitafederal/pt-br/assuntos/noticias/2024/outubro/cnpj-tera-letras-e-numeros-a-partir-de-julho-de-2026
func MatchCNPJV2(_ context.Context, s []byte) bool {
	// Remove formatting characters
	cnpj := cleanCNPJFormat(s)

	// A valid CNPJ must have 14 characters without punctuations
	if len(cnpj) != 14 {
		return false
	}

	// Convert to uppercase for alphanumeric validation
	cnpjUpper := bytes.ToUpper(cnpj)

	// Check if it contains at least one letter (new alphanumeric format requirement)
	hasLetters := bytes.IndexFunc(cnpjUpper, func(r rune) bool {
		return (r >= 'A' && r <= 'Z')
	}) != -1

	if !hasLetters {
		return false
	}

	// Validate against the alphanumeric pattern
	if !cnpjAlphanumRegexp.Match(cnpjUpper) {
		return false
	}

	numericValues := cnpjAlphaToNumeric(cnpjUpper)

	d1 := checksum(numericValues, cnpjFirstDigitTable)
	d2 := checksum(numericValues, cnpjSecondDigitTable)

	// Compare calculated check digits with the last two characters
	return numericValues[12] == d1 && numericValues[13] == d2
}

func checksum(ds []int, ref []int) int {
	var s int
	for i, n := range ref {
		s += n * ds[i]
	}

	r := s % 11
	if r < 2 {
		return 0
	}

	return 11 - r
}

func byteAtoi(b byte) (int, error) {
	if b < '0' || b > '9' {
		return 0, fmt.Errorf("invalid byte for conversion: %c", b)
	}

	return int(b - '0'), nil
}
