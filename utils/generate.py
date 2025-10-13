#!/usr/bin/env python3

import os
import time
import requests
import sys

def main():
    url = os.getenv("URL")
    port = os.getenv("PORT")
    
    if not url or not port:
        print("ERROR: URL and PORT environment variables must be set")
        sys.exit(1)
    
    api_endpoint = f"http://{url}:{port}/message"
    quotes_file = os.path.join(os.path.dirname(__file__), "quotes")
    
    print(f"Starting quote generator - posting to {api_endpoint} every 15 seconds")
    
    with open(quotes_file, "r") as f:
        quotes = [line.strip() for line in f if line.strip()]
    
    quote_index = 0
    
    while True:
        quote = quotes[quote_index % len(quotes)]
        
        payload = {
            "original_text": quote,
            "modified_text": ""
        }
        
        try:
            response = requests.post(api_endpoint, json=payload)
            print(f"Posted quote {quote_index + 1}: {quote[:50]}... - Status: {response.status_code}")
        except Exception as e:
            print(f"Error posting quote: {e}")
        
        quote_index += 1
        time.sleep(15)

if __name__ == "__main__":
    main()

