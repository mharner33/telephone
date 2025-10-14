import os
import time
import requests
import sys
import logging

logging.basicConfig(
    level=os.getenv("LOG_LEVEL", "INFO"),
    format="%(asctime)s %(levelname)s %(message)s",
)

def main():
    url = os.getenv("URL")
    port = os.getenv("PORT")
    
    if not url or not port:
        logging.error("URL and PORT environment variables must be set")
        sys.exit(1)
    
    api_endpoint = f"http://{url}:{port}/message"
    quotes_file = os.path.join(os.path.dirname(__file__), "quotes")
    
    logging.info(f"Starting quote generator - posting to {api_endpoint} every 15 seconds")
    
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
            response = requests.post(api_endpoint, json=payload, timeout=10)
            logging.info(f"Posted quote {quote_index + 1}: {quote[:50]}... - Status: {response.status_code}")
        except Exception as e:
            logging.exception("Error posting quote")
        
        quote_index += 1
        time.sleep(15)

if __name__ == "__main__":
    main()
