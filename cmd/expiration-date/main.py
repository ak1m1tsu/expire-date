import os
import re
import pytesseract
import pandas as pd

from PIL import Image
from datetime import datetime


VALID = "VALID"
INVALID = "INVALID"
UNDEFINED = "UNDEFINED"


class Case:
    def __init__(self, path: str, date: datetime = None, status: str = None):
        self.path = path
        self.date = date
        self.status = status if status else UNDEFINED

    def __str__(self) -> str:
        return f"<Case:{self.path},{self.date},{self.status}>"


def get_test_cases(file: str) -> list:
    test_cases = []
    df = pd.read_csv(os.path.abspath(file))
    for i, row in df.iterrows():
        date = row["date"][: len(row["date"]) - 7]
        test_cases.append(
            Case(
                row["path"],
                datetime.strptime(
                    date,
                    "%Y-%m-%d %H:%M:%S",
                ),
            )
        )
    return test_cases


def main():
    find_dates = re.compile(
        r"\b(?:0[1-9]|[1-2][0-9]|3[01])[\.\/](?:0[1-9]|1[0-2])[\.\/](?:\d{4}|\d{2})\b"
    )
    testdata_folder = os.path.abspath("../../test/data")
    files = [
        os.path.join(testdata_folder, file)
        for file in os.listdir(testdata_folder)
        if os.path.isfile(os.path.join(testdata_folder, file))
    ]
    test_cases = get_test_cases("expire-dates.csv")
    actual_results = []
    valid_cases = []
    invalid_cases = []
    undefined_cases = []
    lang = "rus"
    start = datetime.now()

    for file in list(sorted(files)):
        cs = Case(file)
        actual_results.append(cs)
        text = str(pytesseract.image_to_string(Image.open(file), lang=lang))
        clean_text = text.replace("\n", " ")
        matches = find_dates.findall(clean_text)
        if not matches:
            undefined_cases.append(cs)
            continue

        dates = []
        for match in matches:
            match = match.replace("/", ".")
            pieces = match.split(".")
            if len(pieces[2]) == 2:
                pieces[2] = str(datetime.now().year)[:2] + pieces[2]
                match = ".".join(pieces)
            match = match + " 00:00:00"
            try:
                date = datetime.strptime(match, "%d.%m.%Y %H:%M:%S")
                dates.append(date)
            except ValueError:
                pass

        cs.status = INVALID
        match len(dates):
            case 1:
                cs.date = dates[0]
            case 2:
                if dates[0] < dates[1]:
                    cs.date = dates[1]
                else:
                    cs.date = dates[0]

    for k, v in enumerate(actual_results):
        if v.status == UNDEFINED:
            continue
        if test_cases[k].date == v.date:
            v.status = VALID
            valid_cases.append(v)
            continue
        invalid_cases.append(v)
    print("Done in:", datetime.now() - start)
    print_cases(VALID, valid_cases)
    print_cases(INVALID, invalid_cases)
    print_cases(UNDEFINED, undefined_cases)


def print_cases(status, cases):
    print(f"{status} cases: {len(cases)}")
    for case in cases:
        print("\t", case.path, "-", case.date)


if __name__ == "__main__":
    main()
