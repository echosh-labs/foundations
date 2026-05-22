#!/usr/bin/env python3
import sys
import json
import socket
import datetime
import argparse

def main():
    parser = argparse.ArgumentParser(description="Broadcast telemetry events to foundations socket")
    parser.add_argument("-t", "--type", default="LOG", help="Event type (LOG, STATE, TASK, METRIC)")
    parser.add_argument("-s", "--state", default="", help="Agent state (IDLE, EXECUTING, HEALING, COMPLETE, etc.)")
    parser.add_argument("-a", "--agent", default="ANTIGRAVITY", help="Agent ID")
    parser.add_argument("-c", "--component", default="", help="Component name")
    parser.add_argument("-v", "--severity", default="INFO", help="Severity (INFO, WARNING, CRITICAL)")
    parser.add_argument("-m", "--metrics", default="", help="JSON string of metrics")
    parser.add_argument("message", nargs="*", help="Log message (reads from stdin if empty)")
    
    args = parser.parse_args()
    
    if args.message:
        message = " ".join(args.message)
    else:
        message = sys.stdin.read().strip()
        
    event = {
        "timestamp": datetime.datetime.now(datetime.timezone.utc).strftime("%Y-%m-%dT%H:%M:%SZ"),
        "type": args.type,
        "message": message,
        "agent_id": args.agent
    }
    if args.state:
        event["state"] = args.state
    if args.component:
        event["component"] = args.component
    if args.severity:
        event["severity"] = args.severity
    if args.metrics:
        try:
            event["metrics"] = json.loads(args.metrics)
        except Exception as e:
            # If it's a key-value comma-separated string, try parsing it as simple floats/ints
            # e.g., stabilization:92.5,flux_density:4.2
            try:
                metrics_dict = {}
                for item in args.metrics.split(','):
                    k, v = item.split(':')
                    try:
                        metrics_dict[k.strip()] = float(v.strip())
                    except ValueError:
                        metrics_dict[k.strip()] = v.strip()
                event["metrics"] = metrics_dict
            except Exception:
                event["metrics"] = {}
        
    payload = json.dumps(event) + "\n"
    
    try:
        with socket.socket(socket.AF_UNIX, socket.SOCK_STREAM) as client:
            client.connect("/tmp/foundations.sock")
            client.sendall(payload.encode("utf-8"))
    except Exception as e:
        # Silently fail if server is not running or socket is missing
        pass

if __name__ == "__main__":
    main()
