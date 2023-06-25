import os
import re
import pytesseract
import pandas as pd

from PIL import Image
from datetime import datetime
from pytz import timezone


VALID = "VALID"
INVALID = "INVALID"
UNDEFINED = "UNDEFINED"


class Case:
    def __init__(self, path, date):
        self.path = path
        self.date = date
        self.status = UNDEFINED


def main():
    start = datetime.now()
    valid_cases = []
    invalid_cases = []
    undefined_cases = []
    find_dates = re.compile(
        r"\b(?:0[1-9]|[1-2][0-9]|3[01])[\.\/](?:0[1-9]|1[0-2])[\.\/](?:\d{4}|\d{2})\b"
    )
    testdata_folder = os.path.abspath("test/data")

    files = [
        os.path.join(testdata_folder, file) for file in os.listdir(testdata_folder)
    ]
    dates = []
    
    
    for file in files:
        text = str(pytesseract.image_to_string(Image.open(file), lang="rus"))
        text = text.replace("\n", " ")
        result = find_dates.findall(text)
        if not result:
            undefined_cases.append(Case(file, None))
            continue
        valid_cases.append(Case(file, result))
        
    
    timesince = datetime.now() - start
    print("Time since start script:", timesince)
    print_cases(VALID, valid_cases)
    print_cases(INVALID, invalid_cases)
    print_cases(UNDEFINED, undefined_cases)

def print_cases(status, cases):
    print(f"{status} cases: {len(cases)}")
    for case in cases:
        print("\t", case.path, "-", case.date)


if __name__ == "__main__":
    main()
