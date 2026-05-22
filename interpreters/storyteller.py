#!/usr/bin/env python3
import os
import sys
import json
import socket
import datetime
import argparse

def broadcast_log(msg, state="EXECUTING", agent="STORYTELLER"):
    event = {
        "timestamp": datetime.datetime.now(datetime.timezone.utc).strftime("%Y-%m-%dT%H:%M:%SZ"),
        "type": "LOG",
        "message": msg,
        "agent_id": agent,
        "state": state
    }
    payload = json.dumps(event) + "\n"
    try:
        with socket.socket(socket.AF_UNIX, socket.SOCK_STREAM) as client:
            client.connect("/tmp/foundations.sock")
            client.sendall(payload.encode("utf-8"))
    except:
        pass

def plan_scenes(content):
    content_lower = content.lower()
    
    # 1. Check for Intuition / Idealism / Illumination keywords
    if "illumination" in content_lower or "intuition" in content_lower or "idealism" in content_lower or "lewis" in content_lower:
        return [
            {
                "id": 1,
                "title": "Intuition: The Inner Staircase",
                "prompt": "Sleek cybernetic figure standing at the base of a glowing neon staircase, soft neon violet aura surrounding their head, projecting faint holographic glyphs, dark moody background, high contrast HSL tailored purple and deep blue tones",
                "narrative": "Our intuition helps us to form a series of steps to climb, a deep inner guidance satisfying our highest psychic self."
            },
            {
                "id": 2,
                "title": "Idealism: The Ascent of Aspiration",
                "prompt": "Sleek cybernetic figure climbing a high glowing neon staircase towards a soaring futuristic geometric light architecture, bright cyan lines, glowing orange ember shadows, cinematic lighting",
                "narrative": "Each step in turn is an ideal, ever more advanced, broadening our consciousness and preparing it for the final breakthrough."
            },
            {
                "id": 3,
                "title": "Illumination: Radiant Consciousness",
                "prompt": "A figure at the summit of a high neon structure, their consciousness expanding as a brilliant golden and peach watercolor nebula, cosmic nebula background, space-age cybernetic style",
                "narrative": "The summit of understanding. Idealism prepares the consciousness, and Illumination follows as a radiant, unified state of being."
            }
        ]

    # 2. Check for Psychocentric keywords
    if "threat filter" in content_lower or "power gauge" in content_lower or "psychocentric" in content_lower:
        return [
            {
                "id": 1,
                "title": "The Threat Filter Eye Scan",
                "prompt": "Futuristic cyberpunk interface overlay, detailed mechanical eye scan HUD, glitch effects, neon cyan and red lines, deep contrast, high fidelity illustration",
                "narrative": "Stop. Your brain thinks you're in danger right now. It's lying. Survival mode hijacks attention."
            },
            {
                "id": 2,
                "title": "The Scrolling Power Gauge",
                "prompt": "Cybernetic digital energy meter indicating attention power gauge, scrolling status HUD, neon green and purple light beams, high tech control panel, sleek cyberpunk aesthetics",
                "narrative": "You're scrolling to win. To find the 'best' info. To be ahead of the curve. Your centers are on autopilot."
            },
            {
                "id": 3,
                "title": "Gut Compass & Mirror Lens",
                "prompt": "Holographic human silhouette with a glowing breaking glass heart, purple and golden light, digital glitch transit, futuristic HUD dashboard, premium vector art",
                "narrative": "Notice the urge to scroll away. That's the glitch. Don't flip the phone. Flip the switch."
            }
        ]
        
    # 2. Check for Foundational Story keywords
    if "foundational story" in content_lower or "healing" in content_lower or "remorse" in content_lower or "luke" in content_lower:
        return [
            {
                "id": 1,
                "title": "Visceral Interconnectedness",
                "prompt": "Ethereal abstract illustration representing deep human interconnectedness, glowing neon blue lines linking silhouettes of diverse people, dark moody background, high contrast, HSL tailored vibrant colors",
                "narrative": "A frightening and visceral perception of profound human interconnectedness, experienced as a storm of sleep-deprived awareness."
            },
            {
                "id": 2,
                "title": "The Path of Remorse & Love",
                "prompt": "Atmospheric digital painting depicting emotional warmth amidst sorrow, a glowing orange fire ember casting long gentle shadows in a cool dark blue room, cinematic lighting",
                "narrative": "A confession of morbid sorrow and profound responsibility, met with an overwhelming, enduring love for Luke."
            },
            {
                "id": 3,
                "title": "Family Healing & Ethereal Light",
                "prompt": "Gentle ethereal watercolor painting of a serene warm sunrise, soft golden and peach light breaking through clouds over a peaceful walking path, familial peace and hope",
                "narrative": "Deep familial healing and reconciliation over the decades, establishing a foundational stone for the future."
            }
        ]
        
    # 3. Fallback generic dynamic scenes
    return [
        {
            "id": 1,
            "title": "Initiation & Concept",
            "prompt": "Sleek futuristic cybernetic startup interface, neon blue highlights, abstract glowing shapes, high contrast digital art, HSL tailored",
            "narrative": f"Exploring the core concepts and initiating the workflow for: {content[:60]}..."
        },
        {
            "id": 2,
            "title": "System Architecture",
            "prompt": "Detailed technical machinery blueprint HUD, glowing orange and cyan vectors, high fidelity vector graphics, dark moody backdrop",
            "narrative": "Interpreting complex requirements and constructing a clean, scalable technical framework."
        },
        {
            "id": 3,
            "title": "Realization & Launch",
            "prompt": "Bright golden sunset over a high-tech modern cityscape, soft reflections, smooth clean lines, premium visual finish",
            "narrative": "Bringing the design system to life and deploying the finished high-fidelity interactive digital asset."
        }
    ]

def setup_webapp_boilerplate(dest_dir, title, scenes):
    os.makedirs(dest_dir, exist_ok=True)
    os.makedirs(os.path.join(dest_dir, "assets"), exist_ok=True)
    
    # Save the scenes.json metadata
    with open(os.path.join(dest_dir, "scenes.json"), "w") as f:
        json.dump(scenes, f, indent=2)
        
    broadcast_log(f"Created metadata scenes.json in {dest_dir}")

def main():
    parser = argparse.ArgumentParser(description="Orchestrate and plan creative story publishing")
    parser.add_argument("-t", "--title", default="Untitled Story", help="Story title")
    parser.add_argument("-o", "--output", default="/home/justin/code/echosh-labs/foundations/web-app", help="Output directory for web application")
    parser.add_argument("content", nargs="*", help="Story content (reads from stdin if empty)")
    
    args = parser.parse_args()
    
    if args.content:
        content = " ".join(args.content)
    else:
        content = sys.stdin.read().strip()
        
    broadcast_log(f"Starting orchestration for: {args.title}")
    
    # Plan visual storyboard scenes
    broadcast_log("Analyzing narrative structure and planning storyboard scenes...")
    scenes = plan_scenes(content)
    for s in scenes:
        broadcast_log(f"Outlined Scene {s['id']}: {s['title']} -> '{s['prompt']}'")
        
    # Setup web-app folders and metadata
    broadcast_log(f"Setting up publishing directories under {args.output}...")
    setup_webapp_boilerplate(args.output, args.title, scenes)
    
    broadcast_log("Orchestration and storytelling plan created successfully!", state="COMPLETE")
    
    print(json.dumps({
        "title": args.title,
        "scenes": scenes,
        "output_dir": args.output
    }, indent=2))

if __name__ == "__main__":
    main()
