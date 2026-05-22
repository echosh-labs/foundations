#!/usr/bin/env python3
import sys
import json
import argparse
import re

def classify_content(title, content):
    title_lower = title.lower()
    content_lower = content.lower()
    
    # 1. Explicit Title Overrides (Highest Precedence)
    story_prefixes = ["storyboard:", "foundational story:", "script:", "scene:", "sorry board:"]
    code_prefixes = ["system prompt:", "software package:", "technical instructions:", "integration:", "[audio]", "tui:"]
    strategy_prefixes = ["strategic plan:", "roadmap:", "budget:", "policy:"]
    
    if any(title_lower.startswith(p) for p in story_prefixes) or "sorry board" in title_lower or "storyboard" in title_lower:
        return "CREATIVE_STORY"
    if any(title_lower.startswith(p) for p in code_prefixes):
        return "CODE_DEV"
    if any(title_lower.startswith(p) for p in strategy_prefixes):
        return "STRATEGIC_PLAN"
        
    # 2. Weighted Keyword Scoring
    keywords = {
        "CREATIVE_STORY": {
            "story": 2.0, "narrative": 2.0, "healing": 2.5, "remorse": 2.5, 
            "luke": 3.0, "revelation": 2.0, "script": 1.5, "scene": 1.5, 
            "storyboard": 2.0, "watercolor": 2.0, "illustration": 2.0, 
            "painting": 2.0, "cinematic": 1.5, "video": 1.0, "glitch-effect": 2.0,
            "character": 1.0, "desolate": 1.0, "beat": 1.0, "dialogue": 2.0
        },
        "CODE_DEV": {
            "code": 2.0, "golang": 2.5, "compile": 2.5, "build": 1.5, 
            "binary": 2.0, "tui": 3.0, "mcp": 3.0, "bug": 2.0, 
            "refactor": 2.0, "module": 1.5, "go.mod": 3.0, "telemetry": 2.5, 
            "dashboard": 2.0, "subsystem": 2.0, "api": 2.0, "rest": 1.5, 
            "endpoint": 2.0, "backend": 2.0, "frontend": 2.0, "ui": 1.5, 
            "unit test": 2.5, "logger": 1.5, "socket": 2.0, "integration": 1.5, 
            "server": 1.5, "database": 2.0, "sqlite": 2.5, "wsl": 2.0, 
            "named pipes": 2.5, "microservice": 2.5, "git": 1.5, "github": 1.5, 
            "repo": 1.5, "repository": 1.5, "asynchronously": 1.5, "non-blocking": 1.5
        },
        "STRATEGIC_PLAN": {
            "budget": 2.0, "strategic plan": 3.0, "roadmap": 2.0, 
            "invoice": 2.5, "finance": 2.0, "email": 1.0, "admin": 1.5, 
            "sheet": 1.5, "corporate": 1.5, "letter to": 2.0, "document": 1.0, 
            "policy": 2.0, "reconciliation": 1.0, "business": 1.5, 
            "starter": 1.5, "authorize": 1.0
        }
    }
    
    scores = {"CREATIVE_STORY": 0.0, "CODE_DEV": 0.0, "STRATEGIC_PLAN": 0.0}
    
    # Score the title (keywords have triple weight in title)
    for domain, kw_map in keywords.items():
        for kw, weight in kw_map.items():
            pattern = r'\b' + re.escape(kw) + r'\b'
            if re.search(pattern, title_lower):
                scores[domain] += weight * 3.0
                
    # Score the content body
    for domain, kw_map in keywords.items():
        for kw, weight in kw_map.items():
            pattern = r'\b' + re.escape(kw) + r'\b'
            count = len(re.findall(pattern, content_lower))
            scores[domain] += count * weight
            
    # Find the domain with the highest score
    max_domain = max(scores, key=scores.get)
    
    # If the highest score is very low (e.g. less than 2.0), default to GENERAL_TASK
    if scores[max_domain] < 2.0:
        return "GENERAL_TASK"
        
    return max_domain

def main():
    parser = argparse.ArgumentParser(description="Classify Keep note content into an agentic domain")
    parser.add_argument("-t", "--title", default="Untitled", help="Keep note title")
    parser.add_argument("content", nargs="*", help="Keep note content (reads from stdin if empty)")
    
    args = parser.parse_args()
    
    if args.content:
        content = " ".join(args.content)
    else:
        content = sys.stdin.read().strip()
        
    domain = classify_content(args.title, content)
    
    result = {
        "title": args.title,
        "domain": domain,
        "length": len(content)
    }
    
    print(json.dumps(result, indent=2))

if __name__ == "__main__":
    main()
