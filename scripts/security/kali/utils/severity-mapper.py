#!/usr/bin/env python3
# scripts/security/kali/utils/severity-mapper.py

import json
import argparse
import sys

def check_thresholds(report_path):
    try:
        with open(report_path, 'r') as f:
            report = json.load(f)
        
        summary = report.get('summary', {}).get('by_severity', {})
        
        critical = summary.get('CRITICAL', 0)
        high = summary.get('HIGH', 0)
        
        # Hardcoded thresholds matching scan-targets.yaml for now
        # Ideally we would read yaml config here too
        
        if critical > 0:
            print(f"FAILED: Found {critical} CRITICAL vulnerabilities")
            return 1
            
        if high > 0:
            print(f"FAILED: Found {high} HIGH vulnerabilities")
            return 1
            
        print("PASS: Severity thresholds met")
        return 0
        
    except Exception as e:
        print(f"Error checking thresholds: {e}")
        return 1

def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--report", required=True)
    parser.add_argument("--check-thresholds", action="store_true")
    
    args = parser.parse_args()
    
    if args.check_thresholds:
        sys.exit(check_thresholds(args.report))

if __name__ == "__main__":
    main()
