# BrosyncDelivery

> **Educational / Research tool — use responsibly**

This project is intended **only** for educational and defensive research (red-team / blue-team) to help security professionals understand stealthy data delivery and exfiltration techniques so they can improve detection and mitigation controls. Do **not** use this tool against systems you do not own or have explicit permission to test. Refer to your local laws and organizational policies before running any tests.

---

## Overview

BrosyncDelivery demonstrates a stealthy technique that leverages Brave browser's history sync to deliver or transfer small chunks of data across machines. The goal of this proof-of-concept is to help defenders and researchers study: detection gaps, forensic artifacts, and mitigation strategies related to browser sync features.

**Important:** This project focuses on Brave's sync mechanism because it can synchronize history between browsers using a sync code rather than a full user account. This POC is meant to illustrate the concept — it is not a production exfiltration tool.

---

## Features

* Split input data into small, URL-like chunks and write them into Brave's history database.
* Option to open Brave after writing entries (Windows default behaviour).
* Tools to encode (write) and decode (read) payloads to/from the Brave history DB.

---

## Requirements

* Target platform: Windows (compiled binary or run with appropriate environment).
* Brave browser installed on the machine where you intend to write or read the history file.
* Appropriate permissions to read/write the Brave history SQLite file for the user profile being targeted.

---

## Installation

1. Clone the repository:

```bash
git clone <your-repo-url>
cd BrosyncDelivery
```

2. Build (example using Go, if this is a Go project):

```bash
go build -o BrosyncDelivery.exe .
```

Adjust build steps for your environment and toolchain.

---

## Usage

**Encode (embed data into Brave history):**

```powershell
# encode <input-file>
.\BrosyncDelivery.exe encode <input-file>
```

By default this will write to the current user's Brave history database by attempting to open Brave.

To specify a custom Brave binary path (Windows):

```powershell
.\BrosyncDelivery.exe encode <input-file> "C:\Path\To\brave.exe"
```

**Decode (extract data from Brave history):**

```powershell
# decode <output-folder>
.\BrosyncDelivery.exe decode <output-folder>
```

By default the tool looks for the current user's Brave history file. To specify a custom history file path:

```powershell
.\BrosyncDelivery.exe decode <output-folder> "C:\Path\To\History"
```

> Note: Replace `.\BrosyncDelivery.exe` with your built binary name or run via your development environment if appropriate.

---

## How it works (high level)

1. The tool reads the input file and encodes it into URL-like strings (small chunks) suitable for insertion into Brave's `History`/`urls` table.
2. When Brave syncs across devices using t
