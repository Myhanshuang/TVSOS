import requests
import json
import time

url="http://localhost:8080/api/vehicles/location/report"

data = {
            "license": "川A12345",
            "lon": 102.123456,
            "lat": 25.123456,
            "speed": 0.00,
            "timestamp": time.time()
        }

json_data = json.dumps(data)
headers = {'Content-Type': 'application/json'}

response = requests.post(url, data=json_data, headers=headers)
print(response.status_code, response.text)