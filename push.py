#!/usr/bin/env python3
import subprocess
subprocess.run(["git", "add", "."])
subprocess.run(["git", "commit", "-m", '"update"'])
subprocess.run(["git", "push"])
