package entities

import (
	"github.com/threatwinds/go-sdk/utils"
)

const (
	STR         = "Case-sensitive string"
	IP          = "IP"
	EMAIL       = "Email"
	FQDN        = "FQDN"
	INTEGER     = "Integer"
	CIDR        = "CIDR"
	CITY        = "City"
	COUNTRY     = "Country"
	FLOAT       = "Float"
	URL         = "URL"
	MD5         = "MD5"
	HEXADECIMAL = "Hexadecimal"
	BASE64      = "BASE64"
	DATE        = "Date"
	MAC         = "MAC"
	MIME        = "MIME type"
	PHONE       = "Phone"
	SHA1        = "SHA-1"
	SHA224      = "SHA-224"
	SHA256      = "SHA-256"
	SHA384      = "SHA-384"
	SHA512      = "SHA-512"
	SHA3_224    = "SHA3-224"
	SHA3_256    = "SHA3-256"
	SHA3_384    = "SHA3-384"
	SHA3_512    = "SHA3-512"
	SHA512_224  = "SHA512-224"
	SHA512_256  = "SHA512-256"
	DATETIME    = "Datetime"
	UUID        = "UUID"
	BOOLEAN     = "Boolean"
	ISTR        = "String"
	PATH        = "Path"
	IDENTIFIER  = "UUID|MD5|SHA3-256"
	ADVERSARY   = "Adversary"
	REGEX       = "Regex"
	PORT        = "Port"
)

type Definition struct {
	Type         string       `json:"type" example:"object"`
	Description  string       `json:"description" example:"Important description about the type"`
	DataType     string       `json:"dataType" example:"String"`
	Example      *Entity      `json:"example,omitempty"`
	Attributes   []Definition `json:"attributes,omitempty"`
	Associations []Definition `json:"associations,omitempty"`
	Tags         []string     `json:"tags,omitempty"`
	Correlate    []string     `json:"correlate,omitempty"`
	Label        string       `json:"label,omitempty"`
}

var file = Definition{
	Type:         "file",
	Description:  "An object identifying a file, the value can be a UUID or a SHA3-256 or MD5 checksum",
	DataType:     IDENTIFIER,
	Attributes:   []Definition{fileData, hashSHA1, hashMD5, hashSHA256, hashSHA3256},
	Associations: []Definition{filename, filenamePattern},
	Tags:         []string{"malware", "common-file", "system-file"},
	Correlate:    []string{"md5", "sha1", "sha256", "sha3-256", "file-data"},
	Example:      &eFile,
	Label:        "File",
}

var payload = Definition{
	Type:        "payload",
	Description: "A SHA3-256 of a message sent in a network packet",
	DataType:    SHA3_256,
	Attributes:  []Definition{hashSHA1, hashMD5, hashSHA256, hashSHA3256},
	Label:       "Payload",
}

var fileData = Definition{
	Type:        "file-data",
	Description: "A file or attachment URL",
	DataType:    URL,
	Attributes:  []Definition{},
	Label:       "Download",
}

var adversary = Definition{
	Type:        "adversary",
	Description: "An object identifying a threat actor",
	DataType:    ADVERSARY,
	Label:       "Adversary",
}

var aso = Definition{
	Type:        "aso",
	Description: "An autonomous System Organization",
	DataType:    ISTR,
	Label:       "ASO",
}

var asn = Definition{
	Type:        "asn",
	Description: "An autonomous System Organization Number",
	DataType:    INTEGER,
	Label:       "ASN",
}

var malware = Definition{
	Type:        "malware",
	Description: "A software intentionally designed to harm a computer, server, network, or user",
	DataType:    ISTR,
	Attributes: []Definition{
		malwareFamily,
		malwareType,
	},
	Correlate: []string{"malware-family", "malware-type"},
	Example:   &eMalware,
	Label:     "Malware Name",
}

var abaRtn = Definition{
	Type:        "aba-rtn",
	Description: "An ABA routing transit number",
	DataType:    INTEGER,
	Label:       "ABA RTN",
}

var latitude = Definition{
	Type:        "latitude",
	Description: "A GPS latitude",
	DataType:    FLOAT,
	Label:       "Latitude",
}

var longitude = Definition{
	Type:        "longitude",
	Description: "A GPS longitude",
	DataType:    FLOAT,
	Label:       "Longitude",
}

var accuracyRadius = Definition{
	Type:        "accuracy-radius",
	Description: "A radius of accuracy in meters",
	DataType:    FLOAT,
	Label:       "Accuracy Radius",
}

var country = Definition{
	Type:        "country",
	Description: "A country name",
	DataType:    COUNTRY,
	Label:       "Country",
}

var city = Definition{
	Type:        "city",
	Description: "A city name",
	DataType:    CITY,
	Label:       "City",
}

var cookie = Definition{
	Type:         "cookie",
	Description:  "An HTTP cookie as often stored on the user web client. This can include authentication cookie or session cookie",
	DataType:     STR,
	Associations: []Definition{value},
	Label:        "Cookie",
}

var text = Definition{
	Type:        "text",
	Description: "A case insensitive text value",
	DataType:    ISTR,
	Label:       "Text",
}

var value = Definition{
	Type:        "value",
	Description: "A case sensitive text value",
	DataType:    STR,
	Label:       "Text",
}

var password = Definition{
	Type:        "password",
	Description: "A password",
	DataType:    STR,
	Label:       "Password",
}

var airport = Definition{
	Type:        "airport-name",
	Description: "An airport name",
	DataType:    ISTR,
	Attributes:  []Definition{country, city},
	Label:       "Airport",
}

var profilePhoto = Definition{
	Type:        "profile-photo",
	Description: "A profile photo URL",
	DataType:    URL,
	Label:       "Profile Photo",
}

var authentiHash = Definition{
	Type:        "authentihash",
	Description: "An authenticode executable signature hash",
	DataType:    HEXADECIMAL,
	Label:       "Authenticode Hash",
}

var bankAccountNr = Definition{
	Type:        "bank-account-nr",
	Description: "A bank account number without any routing number",
	DataType:    INTEGER,
	Attributes:  []Definition{bic, bin},
	Label:       "Bank Account",
}

var bic = Definition{
	Type:        "bic",
	Description: "A bank identifier code also known as SWIFT-BIC, SWIFT code or ISO 9362 code",
	DataType:    ISTR,
	Label:       "BIC",
}

var bin = Definition{
	Type:        "bin",
	Description: "A bank identification number",
	DataType:    INTEGER,
	Label:       "BIN",
}

var btc = Definition{
	Type:        "btc",
	Description: "A bitcoin address",
	DataType:    STR,
	Label:       "Bitcoin Address",
}

var ccNumber = Definition{
	Type:        "cc-number",
	Description: "A credit card number",
	DataType:    INTEGER,
	Attributes:  []Definition{issuer},
	Label:       "Credit Card",
}

var issuer = Definition{
	Type:        "issuer",
	Description: "An issuer name",
	DataType:    ISTR,
	Label:       "Issuer",
}

var issuingCountry = Definition{
	Type:        "issuing-country",
	Description: "An issuing country name",
	DataType:    COUNTRY,
	Label:       "Issuing Country",
}

var cdHash = Definition{
	Type:        "cdhash",
	Description: "An Apple Code Directory Hash, identifying a code-signed Mach-O executable file",
	DataType:    HEXADECIMAL,
	Label:       "Code Directory Hash",
}

var certificateFingerprint = Definition{
	Type:        "certificate-fingerprint",
	Description: "A fingerprint of a SSL/TLS certificate",
	DataType:    HEXADECIMAL,
	Label:       "Certificate Fingerprint",
}

var chromeExtension = Definition{
	Type:        "chrome-extension-id",
	Description: "A Chrome extension ID",
	DataType:    STR,
	Label:       "Chrome Extension",
}

var subnet = Definition{
	Type:        "cidr",
	Description: "A public network segment",
	DataType:    CIDR,
	Attributes:  []Definition{country, city, latitude, longitude, asn, aso},
	Label:       "Public Subnet",
}

var cpe = Definition{
	Type:        "cpe",
	Description: "A standardized label used to identify applications, operating systems, and hardware devices",
	DataType:    ISTR,
	Label:       "Platform Enumeration",
}

var cve = Definition{
	Type:        "cve",
	Description: "A standardized identifier for a known cybersecurity vulnerability",
	DataType:    ISTR,
	Label:       "Vulnerability Enumeration",
}

var dash = Definition{
	Type:        "dash",
	Description: "A Dash address",
	DataType:    STR,
	Label:       "Dash",
}

var dkim = Definition{
	Type:        "dkim",
	Description: "A public key used for email authentication",
	DataType:    STR,
	Label:       "DKIM Public Key",
}

var dkimSignature = Definition{
	Type:        "dkim-signature",
	Description: "An email authentication method that helps verify the sender of an email and ensures the message hasn't been tampered with in transit",
	DataType:    STR,
	Label:       "DKIM Signature",
}

var domain = Definition{
	Type:        "domain",
	Description: "A human-readable address that points to a specific website or server on the internet",
	DataType:    FQDN,
	Attributes:  []Definition{whoIsRegistrant, whoIsRegistrar},
	Label:       "Domain",
}

var email = Definition{
	Type:         "email",
	Description:  "An email message ID",
	DataType:     STR,
	Attributes:   []Definition{emailBody, emailDisplayName, emailAddress, emailSubject, emailThreadIndex, emailMimeBoundary, emailXMailer},
	Associations: []Definition{file, emailHeader},
	Label:        "Message ID",
}

var emailAddress = Definition{
	Type:        "email-address",
	Description: "An email sender address",
	DataType:    EMAIL,
	Label:       "Email",
}

var emailBody = Definition{
	Type:        "email-body",
	Description: "An email message body",
	DataType:    ISTR,
	Label:       "Email Body",
}

var emailDisplayName = Definition{
	Type:        "email-display-name",
	Description: "An email sender display name",
	DataType:    ISTR,
	Label:       "Sender Display Name",
}

var emailHeader = Definition{
	Type:        "email-header",
	Description: "An email header",
	DataType:    STR,
	Label:       "Email Header",
}

var emailMimeBoundary = Definition{
	Type:        "email-mime-boundary",
	Description: "A strings of 7-bit US-ASCII text that define the boundaries between message parts in a MIME message. MIME boundaries are declared in a Content-Type message header for any message that encapsulates more than one message part and in part headers for those parts that encapsulate nested parts.",
	DataType:    STR,
	Label:       "Email Mime Boundary",
}

var emailSubject = Definition{
	Type:        "email-subject",
	Description: "An email subject",
	DataType:    ISTR,
	Label:       "Email Subject",
}

var emailThreadIndex = Definition{
	Type:        "email-thread-index",
	Description: "An email thread index",
	DataType:    BASE64,
	Label:       "Email Thread Index",
}

var emailXMailer = Definition{
	Type:        "email-x-mailer",
	Description: "An email x-mailer header",
	DataType:    ISTR,
	Label:       "Email X-Mailer",
}

var eppn = Definition{
	Type:        "eppn",
	Description: "A NetId of a person for the purposes of inter-institutional authentication. Should be stored in the form of user@univ.edu, where univ.edu is the name of the local security domain",
	DataType:    EMAIL,
	Label:       "EPPN",
}

var facebookProfile = Definition{
	Type:        "facebook-profile",
	Description: "A Facebook profile",
	DataType:    URL,
	Label:       "Facebook Profile",
}

var ffn = Definition{
	Type:        "ffn",
	Description: "A frequent flyer number of a passenger",
	DataType:    STR,
	Label:       "Frequent Flyer Number",
}

var filename = Definition{
	Type:        "filename",
	Description: "A filename or email attachment name",
	DataType:    ISTR,
	Label:       "File Name",
}

var sizeInBytes = Definition{
	Type:        "size-in-bytes",
	Description: "A size in bytes of an element",
	DataType:    FLOAT,
	Label:       "Size (Bytes)",
}

var filenamePattern = Definition{
	Type:        "filename-pattern",
	Description: "A pattern in the name of a file",
	DataType:    REGEX,
	Label:       "Filename Pattern",
}

var flight = Definition{
	Type:        "flight",
	Description: "A flight number",
	DataType:    STR,
	Label:       "Flight Number",
}

var gitHubOrganization = Definition{
	Type:        "github-organization",
	Description: "A Github organization",
	DataType:    URL,
	Label:       "GitHub Organization",
}

var gitHubRepository = Definition{
	Type:        "github-repository",
	Description: "A Github repository",
	DataType:    URL,
	Label:       "GitHub Repo",
}

var gitHubUser = Definition{
	Type:        "github-user",
	Description: "A Github profile",
	DataType:    URL,
	Label:       "GitHub Profile",
}

var link = Definition{
	Type:        "link",
	Description: "An external link for reference",
	DataType:    URL,
	Label:       "Reference",
}

var datetime = Definition{
	Type:        "datetime",
	Description: "A time with nanoseconds in the format 2006-01-02T15:04:05.999999999Z07:00",
	DataType:    DATETIME,
	Label:       "Date & Time",
}

var lastAnalysis = Definition{
	Type:        "last-analysis",
	Description: "A time of last analysis. Format 2006-01-02T15:04:05.999999999Z",
	DataType:    DATETIME,
	Label:       "Last Analysis",
}

var date = Definition{
	Type:        "date",
	Description: "A date in format 2006-01-02",
	DataType:    DATE,
	Label:       "Date",
}

var dateOfIssue = Definition{
	Type:        "date-of-issue",
	Description: "A date in format 2006-01-02",
	DataType:    DATE,
	Label:       "Date of Issue",
}

var expirationDate = Definition{
	Type:        "expiration-date",
	Description: "A date in format 2006-01-02",
	DataType:    DATE,
	Label:       "Due Date",
}

var malwareSample = Definition{
	Type:        "malware-sample",
	Description: "A malware sample URL",
	DataType:    URL,
	Label:       "Malware Sample",
}

var group = Definition{
	Type:        "group",
	Description: "A group of adversaries like APTnn or Anonymous",
	DataType:    ADVERSARY,
	Label:       "Group",
}

var hexa = Definition{
	Type:        "hex",
	Description: "A value in hexadecimal",
	DataType:    HEXADECIMAL,
	Label:       "Hexadecimal",
}

var base64d = Definition{
	Type:        "base64",
	Description: "A value in BASE64 format",
	DataType:    BASE64,
	Label:       "Base64",
}

var hostname = Definition{
	Type:        "hostname",
	Description: "A human-readable name that identifies that specific device, making it easier to locate and interact with it on the network",
	DataType:    FQDN,
	Label:       "Host",
}

var iban = Definition{
	Type:        "iban",
	Description: "An international bank account number",
	DataType:    ISTR,
	Label:       "IBAN",
}

var idNumber = Definition{
	Type:        "id-number",
	Description: "It can be an ID card, residence permit, etc.",
	DataType:    STR,
	Attributes:  []Definition{issuer, dateOfIssue, expirationDate},
	Label:       "Identity Number",
}

var ipAddr = Definition{
	Type:        "ip",
	Description: "An unique numerical identifier assigned to every device connected to a network that uses the internet protocol. It's like a digital address that helps deliver information to the right place on the internet",
	DataType:    IP,
	Attributes:  []Definition{subnet},
	Label:       "IP",
}

var ja3Fingerprint = Definition{
	Type:        "ja3-fingerprint",
	Description: "An SSL/TLS client fingerprints that is easy to produce on any platform and can be easily shared for threat intelligence",
	DataType:    MD5,
	Label:       "JA3 Fingerprint",
}

var jabberID = Definition{
	Type:        "jabber-id",
	Description: "A Jabber ID",
	DataType:    EMAIL,
	Label:       "Jabber ID",
}

var jarmFingerprint = Definition{
	Type:        "jarm-fingerprint",
	Description: "An SSL/TLS server fingerprints",
	DataType:    HEXADECIMAL,
	Label:       "JARM Fingerprint",
}

var macAddr = Definition{
	Type:        "mac-address",
	Description: "A network interface hardware address",
	DataType:    MAC,
	Label:       "Hardware Address",
}

var malwareFamily = Definition{
	Type:        "malware-family",
	Description: "A malware family",
	DataType:    ISTR,
	Label:       "Malware Family",
}

var malwareType = Definition{
	Type:        "malware-type",
	Description: "A malware type",
	DataType:    ISTR,
	Label:       "Malware Type",
}

var hashMD5 = Definition{
	Type:        "md5",
	Description: "A 128-bit fingerprint",
	DataType:    MD5,
	Label:       "MD5",
}

var mimeType = Definition{
	Type:        "mime-type",
	Description: "A two-part identifier",
	DataType:    MIME,
	Label:       "Media Type",
}

var mobileAppID = Definition{
	Type:        "mobile-app-id",
	Description: "An ID of a mobile application",
	DataType:    STR,
	Label:       "Mobile App ID",
}

var passport = Definition{
	Type:        "passport",
	Description: "A passport number",
	DataType:    STR,
	Attributes:  []Definition{issuingCountry, issuer, dateOfIssue, expirationDate},
	Label:       "Passport Number",
}

var pathD = Definition{
	Type:        "path",
	Description: "A path to a file, folder or process, also an HTTP request path",
	DataType:    PATH,
	Label:       "Path",
}

var patternInFile = Definition{
	Type:        "pattern-in-file",
	Description: "A pattern inside a file",
	DataType:    REGEX,
	Label:       "Pattern in File",
}

var patternInMemory = Definition{
	Type:        "pattern-in-memory",
	Description: "A pattern in memory",
	DataType:    REGEX,
	Label:       "Pattern in Memory",
}

var patternInTraffic = Definition{
	Type:        "pattern-in-traffic",
	Description: "A pattern in traffic",
	DataType:    REGEX,
	Label:       "Pattern in Traffic",
}

var pgpPrivateKey = Definition{
	Type:        "pgp-private-key",
	Description: "A private key of a popular encryption system that uses a combination of different cryptographic techniques",
	DataType:    STR,
	Label:       "PGP Private Key",
}

var pgpPublicKey = Definition{
	Type:        "pgp-public-key",
	Description: "A public key of a popular encryption system that uses a combination of different cryptographic techniques",
	DataType:    STR,
	Label:       "PGP Public Key",
}

var phone = Definition{
	Type:        "phone",
	Description: "A phone number",
	DataType:    PHONE,
	Label:       "Phone Number",
}

var pnr = Definition{
	Type:        "pnr",
	Description: "A Passenger Name Record Locator is a key under which the reservation for a trip is stored in the system. The PNR contains, among other data, the name, flight segments and address of the passenger. It is defined by a combination of five or six letters and numbers",
	DataType:    STR,
	Label:       "Reservation Number",
}

var process = Definition{
	Type:        "process",
	Description: "A running process",
	DataType:    ISTR,
	Attributes:  []Definition{processState},
	Label:       "Process",
}

var processState = Definition{
	Type:        "process-state",
	Description: "A state of a process",
	DataType:    ISTR,
	Label:       "Process State",
}

var pRtn = Definition{
	Type:        "prtn",
	Description: "A premium-rate phone number",
	DataType:    ISTR,
	Label:       "PRTN",
}

var redress = Definition{
	Type:        "redress-number",
	Description: "A Redress Control Number is the record identifier for people who apply for redress through the DHS Travel Redress Inquiry Program (DHS TRIP). DHS TRIP is for travelers who have been repeatedly identified for additional screening and who want to file an inquiry to have erroneous information corrected in DHS systems",
	DataType:    STR,
	Label:       "Redress Control Number",
}

var regKey = Definition{
	Type:        "regkey",
	Description: "A registry key",
	DataType:    ISTR,
	Label:       "Registry Key",
}

var hashSHA1 = Definition{
	Type:        "sha1",
	Description: "A 160-bit fingerprint",
	DataType:    SHA1,
	Label:       "SHA-1",
}

var hashSHA224 = Definition{
	Type:        "sha224",
	Description: "A 224-bit fingerprint",
	DataType:    SHA224,
	Label:       "SHA-224",
}

var hashSHA256 = Definition{
	Type:        "sha256",
	Description: "A 256-bit fingerprint",
	DataType:    SHA256,
	Label:       "SHA-256",
}

var hashSHA384 = Definition{
	Type:        "sha384",
	Description: "A 384-bit fingerprint",
	DataType:    SHA384,
	Label:       "SHA-384",
}

var hashSHA512 = Definition{
	Type:        "sha512",
	Description: "A 512-bit fingerprint",
	DataType:    SHA512,
	Label:       "SHA-512",
}

var hashSHA3224 = Definition{
	Type:        "sha3-224",
	Description: "A 224-bit fingerprint",
	DataType:    SHA3_224,
	Label:       "SHA3-224",
}

var hashSHA3256 = Definition{
	Type:        "sha3-256",
	Description: "A 256-bit fingerprint",
	DataType:    SHA3_256,
	Label:       "SHA3-256",
}

var hashSHA3384 = Definition{
	Type:        "sha3-384",
	Description: "A 384-bit fingerprint",
	DataType:    SHA3_384,
	Label:       "SHA3-384",
}

var hashSHA3512 = Definition{
	Type:        "sha3-512",
	Description: "A 512-bit fingerprint",
	DataType:    SHA3_512,
	Label:       "SHA3-512",
}

var hashSHA512224 = Definition{
	Type:        "sha512-224",
	Description: "A 224-bit fingerprint",
	DataType:    SHA512_224,
	Label:       "SHA512-224",
}

var hashSHA512256 = Definition{
	Type:        "sha512-256",
	Description: "A 256-bit fingerprint",
	DataType:    SHA512_256,
	Label:       "SHA512-256",
}

var sshFingerprint = Definition{
	Type:        "ssh-fingerprint",
	Description: "A fingerprint of SSH key material",
	DataType:    STR,
	Label:       "SSH Fingerprint",
}

var sshBanner = Definition{
	Type:        "ssh-banner",
	Description: "An SSH Hello Banner",
	DataType:    STR,
	Label:       "SSH Banner",
}

var ssr = Definition{
	Type:        "ssr",
	Description: "A Special Service Request is a function to an airline to provide a particular facility for A Passenger or passengers",
	DataType:    STR,
	Label:       "Special Service Request",
}

var category = Definition{
	Type:        "category",
	Description: "A label to classify things that share certain characteristics or qualities",
	DataType:    ISTR,
	Label:       "Category",
}

var threat = Definition{
	Type:        "threat",
	Description: "A circumstance or event that has the potential to harm a computer system, network, or data. These threats can compromise the confidentiality, integrity, or availability of digital assets",
	DataType:    ISTR,
	Label:       "Threat",
}

var tikTokProfile = Definition{
	Type:        "tiktok-profile",
	Description: "A TikTok user profile",
	DataType:    URL,
	Label:       "TikTok Profile",
}

var twitterProfile = Definition{
	Type:        "twitter-profile",
	Description: "A Twitter/X user profile",
	DataType:    URL,
	Label:       "Twitter/X Profile",
}

var uri = Definition{
	Type:        "url",
	Description: "An address of a specific resource on the internet, such as a webpage, image, video, or document",
	DataType:    URL,
	Label:       "URL",
}

var username = Definition{
	Type:        "username",
	Description: "An unique identifier that can be used to log in to a website, computer system, online service, or application",
	DataType:    ISTR,
	Label:       "Username",
}

var visa = Definition{
	Type:        "visa",
	Description: "A visa number",
	DataType:    STR,
	Label:       "Visa Number",
}

var whoIsRegistrant = Definition{
	Type:        "whois-registrant",
	Description: "A person or organization who has registered a domain name",
	DataType:    ISTR,
	Label:       "Registrant",
}

var whoIsRegistrar = Definition{
	Type:        "whois-registrar",
	Description: "A company or organization accredited by ICANN (Internet Corporation for Assigned Names and Numbers) to register and manage domain names",
	DataType:    ISTR,
	Label:       "Registrar",
}

var windowsScheduledTask = Definition{
	Type:        "windows-scheduled-task",
	Description: "A Windows scheduled task",
	DataType:    ISTR,
	Label:       "Scheduled Task",
}

var windowsServiceDisplayName = Definition{
	Type:        "windows-service-displayname",
	Description: "A Windows Service's display name, not to be confused with the windows-service-name. This is the name that applications will generally display as the service’s name in applications",
	DataType:    ISTR,
	Label:       "Service Display Name",
}

var windowsServiceName = Definition{
	Type:        "windows-service-name",
	Description: "A Windows Service's name. This is the name used internally by Windows. Not to be confused with the windows-service-displayname",
	DataType:    ISTR,
	Label:       "Service Name",
}

var xmr = Definition{
	Type:        "xmr",
	Description: "A Monero address",
	DataType:    STR,
	Label:       "Monero Address",
}

var breach = Definition{
	Type:        "breach",
	Description: "A security breach that resulted in a leak of PII (Personally Identifiable Information) or SPII (Sensitive Personally Identifiable Information)",
	DataType:    UUID,
	Attributes:  []Definition{domain, link, breachDate, breachCount, breachDescription},
	Label:       "Breach",
}

var breachDate = Definition{
	Type:        "breach-date",
	Description: "Date of breach occurrence",
	DataType:    DATE,
	Label:       "Breach Date",
}

var breachCount = Definition{
	Type:        "breach-count",
	Description: "Number of items leaked in the breach",
	DataType:    INTEGER,
	Label:       "Breach Count",
}

var breachDescription = Definition{
	Type:        "breach-description",
	Description: "A detailed description of the breach",
	DataType:    STR,
	Label:       "Breach Description",
}

var postalAddress = Definition{
	Type:        "postal-address",
	Description: "A postal address",
	DataType:    ISTR,
	Label:       "Postal Address",
}

var zipCode = Definition{
	Type:        "zip-code",
	Description: "A number that identifies a specific geographic region also known as Postal Code",
	DataType:    STR,
	Label:       "Zip Code",
}

var port = Definition{
	Type:        "port",
	Description: "A network ports used in combination with IP addresses to deliver traffic to the right application",
	DataType:    PORT,
	Label:       "Port",
}

var os = Definition{
	Type:        "os",
	Description: "A core software that manages a computer's hardware, software resources, and provides an interface for users to interact with the machine",
	DataType:    ISTR,
	Label:       "Operating System",
}

var command = Definition{
	Type:        "command",
	Description: "A specific instruction you type within a CLI to tell the computer to perform a task",
	DataType:    STR,
	Attributes:  []Definition{os},
	Label:       "Command",
}

var Definitions = []Definition{
	command,
	os,
	port,
	zipCode,
	postalAddress,
	breach,
	breachDate,
	breachCount,
	breachDescription,
	file,
	fileData,
	lastAnalysis,
	adversary,
	aso,
	asn,
	malware,
	malwareFamily,
	malwareType,
	malwareSample,
	abaRtn,
	latitude,
	longitude,
	accuracyRadius,
	country,
	cookie,
	text,
	value,
	issuer,
	password,
	airport,
	profilePhoto,
	authentiHash,
	bankAccountNr,
	bic,
	bin,
	btc,
	ccNumber,
	cdHash,
	certificateFingerprint,
	chromeExtension,
	subnet,
	cpe,
	cve,
	dash,
	dkim,
	dkimSignature,
	domain,
	city,
	issuingCountry,
	emailAddress,
	emailBody,
	emailDisplayName,
	emailHeader,
	emailMimeBoundary,
	emailSubject,
	emailThreadIndex,
	emailXMailer,
	email,
	eppn,
	facebookProfile,
	ffn,
	filename,
	sizeInBytes,
	filenamePattern,
	flight,
	gitHubOrganization,
	gitHubRepository,
	gitHubUser,
	link,
	datetime,
	date,
	dateOfIssue,
	expirationDate,
	group,
	hexa,
	base64d,
	hostname,
	iban,
	idNumber,
	ipAddr,
	ja3Fingerprint,
	jabberID,
	jarmFingerprint,
	sshBanner,
	macAddr,
	hashMD5,
	mimeType,
	mobileAppID,
	passport,
	pathD,
	patternInFile,
	patternInMemory,
	patternInTraffic,
	pgpPrivateKey,
	pgpPublicKey,
	phone,
	pnr,
	process,
	processState,
	pRtn,
	redress,
	regKey,
	hashSHA1,
	hashSHA224,
	hashSHA256,
	hashSHA384,
	hashSHA512,
	hashSHA3224,
	hashSHA3256,
	hashSHA3384,
	hashSHA3512,
	hashSHA512224,
	hashSHA512256,
	sshFingerprint,
	ssr,
	category,
	threat,
	tikTokProfile,
	twitterProfile,
	uri,
	username,
	visa,
	whoIsRegistrant,
	whoIsRegistrar,
	windowsScheduledTask,
	windowsServiceDisplayName,
	windowsServiceName,
	xmr,
	payload,
}

var eMalware = Entity{
	Type: "malware",
	Attributes: Attributes{
		Malware:       utils.PointerOf("pdf dropper agent"),
		MalwareFamily: utils.PointerOf("pdf"),
		MalwareType:   utils.PointerOf("dropper"),
	},
	Correlate:  []string{"malware-family", "malware-type"},
	Reputation: -3,
}

var eFile = Entity{
	Type:       "file",
	Reputation: -3,
	Attributes: Attributes{
		File:    utils.PointerOf("21a1610ce915d5d5a8ab5b1f5b6d6715cf4f4e3bc0c868352a175279b1881afe"),
		Md5:     utils.PointerOf("fb92636db83298a4215a2f5ffa2527b1"),
		Sha1:    utils.PointerOf("93a8f022b553f786bf077ff55616350727f8764a"),
		Sha256:  utils.PointerOf("202492bdd391deac6c1e72eba9d039a7c60bcc61f1afa0d85269d8c4c5af1284"),
		Sha3256: utils.PointerOf("21a1610ce915d5d5a8ab5b1f5b6d6715cf4f4e3bc0c868352a175279b1881afe"),
	},
	Associations: []EntityAssociation{
		{Mode: "aggregation", Entity: eMalware},
	},
	Tags:      []string{"malware", "common-file"},
	Correlate: []string{"md5", "sha1", "sha256", "sha3-256"},
}
