package contract

// W3Token is a Mintable + Burnable ERC-20 built-in, deployed by `w3cli token create`.
// Contract: OpenZeppelin v5.5 ERC20 + ERC20Burnable + Ownable, solc 0.8.26, EVM cancun.
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
//	burn(u256)          → 0x42966c68
//	burnFrom(a,u256)    → 0x79cc6790
//	owner()             → 0x8da5cb5b
//	transferOwnership(a)→ 0xf2fde38b
//	renounceOwnership() → 0x715018a6
//	mint(a,u256)        → 0x40c10f19
func init() {
	RegisterBuiltin(BuiltinKind{
		ID:          "w3token",
		Name:        "W3Token (Mintable+Burnable ERC-20)",
		Description: "OpenZeppelin v5 ERC-20 with mint (onlyOwner) and burn. Deployed via `w3cli token create`.",
		ABI:         w3TokenABI,
	})
}

var w3TokenABI = []ABIEntry{
	// ── ERC-20 read ──────────────────────────────────────────────────────────
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
	// ── ERC-20 write ─────────────────────────────────────────────────────────
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
	// ── ERC20Burnable ────────────────────────────────────────────────────────
	{
		Name: "burn", Type: "function",
		Inputs:          []ABIParam{{Name: "value", Type: "uint256"}},
		Outputs:         nil,
		StateMutability: "nonpayable",
	},
	{
		Name: "burnFrom", Type: "function",
		Inputs:          []ABIParam{{Name: "account", Type: "address"}, {Name: "value", Type: "uint256"}},
		Outputs:         nil,
		StateMutability: "nonpayable",
	},
	// ── Ownable ──────────────────────────────────────────────────────────────
	{
		Name: "owner", Type: "function",
		Inputs: nil, Outputs: []ABIParam{{Name: "", Type: "address"}},
		StateMutability: "view",
	},
	{
		Name: "transferOwnership", Type: "function",
		Inputs:          []ABIParam{{Name: "newOwner", Type: "address"}},
		Outputs:         nil,
		StateMutability: "nonpayable",
	},
	{
		Name: "renounceOwnership", Type: "function",
		Inputs:          nil,
		Outputs:         nil,
		StateMutability: "nonpayable",
	},
	// ── Custom: mint ─────────────────────────────────────────────────────────
	{
		Name: "mint", Type: "function",
		Inputs:          []ABIParam{{Name: "to", Type: "address"}, {Name: "amount", Type: "uint256"}},
		Outputs:         nil,
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
	{
		Name:   "OwnershipTransferred",
		Type:   "event",
		Inputs: []ABIParam{{Name: "previousOwner", Type: "address"}, {Name: "newOwner", Type: "address"}},
	},
}
