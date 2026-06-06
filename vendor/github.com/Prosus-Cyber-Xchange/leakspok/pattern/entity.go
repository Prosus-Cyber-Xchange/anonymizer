package pattern

// Entity represents the type of sensitive data detected (e.g., email, CPF, credit card).
type Entity string

const (
	// EntityEmail identifies email address patterns.
	EntityEmail Entity = "EMAIL"
	// EntityCPF identifies Brazilian CPF numbers.
	EntityCPF Entity = "CPF_NUMBER"
	// EntityCNPJ identifies Brazilian CNPJ numbers.
	EntityCNPJ Entity = "CNPJ_NUMBER"
	// EntityIPAddress identifies IPv4 and IPv6 addresses.
	EntityIPAddress Entity = "IP_ADDRESS"
	// EntityCreditCard identifies credit card numbers.
	EntityCreditCard Entity = "CREDIT_CARD"
	// EntityPhone identifies phone numbers.
	EntityPhone Entity = "PHONE"
	// EntityLink identifies URLs and hyperlinks.
	EntityLink Entity = "LINK"
	// EntitySSN identifies US Social Security Numbers.
	EntitySSN Entity = "SSN"
	// EntityAddress identifies street addresses.
	EntityAddress Entity = "ADDRESS"
	// EntityBankInfo identifies banking information (IBANs, routing numbers).
	EntityBankInfo Entity = "BANK_INFO"
	// EntityUUID identifies UUIDs and GUIDs.
	EntityUUID Entity = "UUID"
	// EntityVIN identifies Vehicle Identification Numbers.
	EntityVIN Entity = "VIN"
)
