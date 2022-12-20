#! /bin/bash


LIST_REGISTRATION_IDS_URL="localhost:9091/api/rest/v1/attendees"
AUTH_HEADER="Authorization: Bearer eyJhbGciOiJSUzUxMiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwiZ2xvYmFsIjp7Im5hbWUiOiJKb2huIERvZSIsInJvbGVzIjpbXX0sImlhdCI6MTUxNjIzOTAyMn0.ove6_7BWQRe9HQyphwDdbiaAchgn9ynC4-2EYEXFeVTDADC4P3XYv5uLisYg4Mx8BZOnkWX-5L82pFO1mUZM147gLKMsYlc-iMKXy4sKZPzhQ_XKnBR-EBIf5x_ZD1wpva9ti7Yrvd0vDi8YSFdqqf7R4RA11hv9kg-_gg1uea6sK-Q_eEqoet7ocqGVLu-ghhkZdVLxu9tWJFPNueILWv8vW1Y_u9fDtfOhw7Ugf5ysI9RXiO-tXEHKN2HnFPCkwccnMFt4PJRzU1VoOldz0xzzZRb-j2tlbjLqcQkjMwLEoPQpC4Wbl8DgkaVdTi2aNyH7EbWMynlSOZIYK0AFvQ"

curl -X GET \
    --header "${AUTH_HEADER}" \
    --url "${LIST_REGISTRATION_IDS_URL}" \
    -v
