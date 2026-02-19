#!/usr/bin/env python3
"""
Fix Qwen3-Omni model configuration bug
The issue: 'Qwen3OmniMoeTalkerCodePredictorConfig' object has no attribute 'use_sliding_window'
"""

import json
import os

model_path = "/mnt/data/models/.cache/models--Qwen--Qwen3-Omni-30B-A3B-Instruct"
config_path = os.path.join(model_path, "config.json")

def fix_config():
    print(f"Reading config from: {config_path}")
    
    with open(config_path, 'r') as f:
        config = json.load(f)
    
    # Check if talker_config exists and fix the code_predictor_config
    if 'talker_config' in config:
        talker_config = config['talker_config']
        print("Found talker_config")
        
        if 'code_predictor_config' in talker_config:
            code_pred_config = talker_config['code_predictor_config']
            print(f"Found code_predictor_config: {code_pred_config}")
            
            # Add use_sliding_window if missing
            if 'use_sliding_window' not in code_pred_config:
                print("Adding use_sliding_window = False")
                code_pred_config['use_sliding_window'] = False
                
            # Add sliding_window if missing
            if 'sliding_window' not in code_pred_config:
                print("Adding sliding_window = 72")
                code_pred_config['sliding_window'] = 72
    
    # Backup original config
    backup_path = config_path + '.backup'
    if not os.path.exists(backup_path):
        print(f"Creating backup: {backup_path}")
        os.rename(config_path, backup_path)
    
    # Write fixed config
    print(f"Writing fixed config to: {config_path}")
    with open(config_path, 'w') as f:
        json.dump(config, f, indent=2)
    
    print("✓ Config fixed!")

if __name__ == "__main__":
    fix_config()
