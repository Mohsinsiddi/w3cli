package contract

// erc20 is the standard ERC-20 interface (EIP-20).
// Use --builtin erc20 to interact with any ERC-20 token without supplying an ABI file.
//
// Function selectors:
//
//	name()              → 0x06fdde03
//	symbol()            → 0x95d89b41
//	decimals()          → 0x313ce567
//	totalSupply()       → 0x18160ddd
//	balanceOf(address)  → 0x70a08231
//	allowance(a,a)      → 0xdd62ed3e
//	transfer(a,u256)    → 0xa9059cbb
//	approve(a,u256)     → 0x095ea7b3
//	transferFrom(a,a,u) → 0x23b872dd
func init() {
	RegisterBuiltin(BuiltinKind{
		ID:          "erc20",
		Name:        "ERC-20 Standard Token",
		Description: "Standard ERC-20 interface (EIP-20). Use `--builtin erc20` with any ERC-20 token.",
		ABI:         erc20ABI,
	})
}

var erc20ABI = []ABIEntry{
	// ── Read ─────────────────────────────────────────────────────────────────
	{
		Name: "name", Type: "function",
		Inputs: nil, Outputs: []ABIParam{{Name: "", Type: "string"}},
		StateMutability: "view",
	},
	{
		Name: "symbol", Type: "function",
		Inputs: nil, Outputs: []ABIParam{{Name: "", Type: "string"}},
		StateMutability: "view",
	},
	{
		Name: "decimals", Type: "function",
		Inputs: nil, Outputs: []ABIParam{{Name: "", Type: "uint8"}},
		StateMutability: "view",
	},
	{
		Name: "totalSupply", Type: "function",
		Inputs: nil, Outputs: []ABIParam{{Name: "", Type: "uint256"}},
		StateMutability: "view",
	},
	{
		Name: "balanceOf", Type: "function",
		Inputs:          []ABIParam{{Name: "account", Type: "address"}},
		Outputs:         []ABIParam{{Name: "", Type: "uint256"}},
		StateMutability: "view",
	},
	{
		Name: "allowance", Type: "function",
		Inputs:          []ABIParam{{Name: "owner", Type: "address"}, {Name: "spender", Type: "address"}},
		Outputs:         []ABIParam{{Name: "", Type: "uint256"}},
		StateMutability: "view",
	},
	// ── Write ────────────────────────────────────────────────────────────────
	{
		Name: "transfer", Type: "function",
		Inputs:          []ABIParam{{Name: "to", Type: "address"}, {Name: "value", Type: "uint256"}},
		Outputs:         []ABIParam{{Name: "", Type: "bool"}},
		StateMutability: "nonpayable",
	},
	{
		Name: "approve", Type: "function",
		Inputs:          []ABIParam{{Name: "spender", Type: "address"}, {Name: "value", Type: "uint256"}},
		Outputs:         []ABIParam{{Name: "", Type: "bool"}},
		StateMutability: "nonpayable",
	},
	{
		Name: "transferFrom", Type: "function",
		Inputs:          []ABIParam{{Name: "from", Type: "address"}, {Name: "to", Type: "address"}, {Name: "value", Type: "uint256"}},
		Outputs:         []ABIParam{{Name: "", Type: "bool"}},
		StateMutability: "nonpayable",
	},
	// ── Events ───────────────────────────────────────────────────────────────
	{
		Name:   "Transfer",
		Type:   "event",
		Inputs: []ABIParam{{Name: "from", Type: "address"}, {Name: "to", Type: "address"}, {Name: "value", Type: "uint256"}},
	},
	{
		Name:   "Approval",
		Type:   "event",
		Inputs: []ABIParam{{Name: "owner", Type: "address"}, {Name: "spender", Type: "address"}, {Name: "value", Type: "uint256"}},
	},
}
