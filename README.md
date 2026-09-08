# 📧 EmailPerm

**EmailPerm** is a fast, modern CLI tool written in Go that generates plausible corporate email permutations based on a target's first name, last name, and company domain. 

Beyond simple generation, EmailPerm includes a statistical heuristic engine to score the plausibility of each permutation based on corporate standards, and an **active SMTP verification engine** to silently validate if the email actually exists on the target server.

## Features

* **Heuristic Scoring:** Ranks permutations based on known enterprise, startup, and legacy naming conventions.
* **Active SMTP Verification:** Shakes hands with the domain's Mail Exchange (MX) server via Port 25 to verify addresses (`RCPT TO`) without sending an actual email.
* **Catch-All Detection:** Automatically detects if a domain is a "catch-all" (accepts all prefixes), preventing false positives.
* **Modern CLI UI:** Beautiful terminal outputs with color-coded results and dynamic progress bars.
* **JSON Output:** Easily pipe verified results into other OSINT tools or data pipelines like `jq`.

---

## Installation

### Option 1: Install via `go install` (Recommended)
If you have Go 1.22+ installed, you can install the binary directly to your `$GOPATH/bin`:
```bash
go install https://github.com/Kynetic-Engynes-Platforms/emailperm@latest
```


### Option 2: Build from Source

Clone the repository and compile it manually:
```bash
git clone https://github.com/Kynetic-Engynes-Platforms/emailperm.git
cd emailperm
go mod tidy
go build -o emailperm main.go

# (Optional) Move to your local bin path
sudo mv emailperm /usr/local/bin/
```

Option 3: Download Pre-compiled Binaries

Visit the Releases page to download pre-compiled binaries for Windows, macOS, and Linux.


## Usage & Use Cases

```bash
emailperm -n <"Full Name"> -d <domain> [options]
```

(Wrap the name in quotes to ensure it is processed as a single string)

1. Multi-Part Name Generation (Offline)

Generate and rank plausible permutations for complex names. This mode dynamically includes middle initials (e.g., kmwangi@) if 3 or more names are provided.
```bash

emailperm -n "Kuria Mwangi Weru" -d kyneticengynes.com
```

2. Active SMTP Verification

Add the -v or --verify flag to actively connect to the domain's MX servers and check which permutations are registered.
```bash
emailperm -n "Kuria Mwangi Weru" -d kyneticengynes.com --verify
```

Note: This will display a live progress bar. Valid emails will be highlighted in green, rejected ones in red. Catch-all domains will trigger a warning and highlight in yellow.

3. Pipeline Integration (JSON Output)

For scripting or chaining with other tools, use the -j or --json flag. This suppresses the UI elements and outputs raw, structured JSON.
```bash

emailperm -n "Jane Doe" -d kyneticengynes.com -v --json > results.json
```

Example piping with jq to extract only valid emails:
```bash
emailperm -n "Kuria Mwangi Weru" -d kyneticengynes.com -v -j | jq -r '.[] | select(.smtp_status == "Valid (250 OK)") | .email'
```

## How the Heuristic Scoring Works

If you run the tool without SMTP verification, it relies on these base probabilities:


| Pattern | Example | Base Score | Context |
| :--- | :--- | :--- | :--- |
| first last | jane doe | 0.90 | Enterprise standard (56% of large corporations) |
| flast | jdoe | 0.75 | Dominant in legacy corporate and tech infrastructure |
| firstlast | janedoe | 0.60 | Common alternative for mid-sized firms |
| first | jane | 0.45 | Dominant in small startups (<20 employees) |



## Troubleshooting

"MX connection failed (ISP port 25 block?)"
Many residential Internet Service Providers (ISPs) silently drop outbound traffic on Port 25 to prevent spam. If the verification hangs or times out, you will need to run this tool on a VPS or through a VPN that permits Port 25 traffic.

