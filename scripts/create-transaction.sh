#! /bin/bash

set -euo pipefail

function _usage() {
    cat <<-EOF
        Helper script to ease up creating transactions for testing purposes

        Usage:  create-transactions.sh [OPTIONS]

        Options:
        -a, --api-key               Create transaction using API key header. Default choice
        -j, --jwt                   Create transaction using predefined JWT
        -d, --debitor int           Provide a custom debitor ID
        -i, --transaction           Provide a custom transaction ID
        -m, --method string         Method (credit, paypal, transfer, internal, gift). Default "credit"
        -t, --type                  Transaction type: (due, payment). Default "due"
        -s, --status                Transaction status: ('tentative', 'pending', 'valid', 'deleted'). Default valid
        -c, --cent                  Gross cent amount. Default 1400
        -h, --help                  Show this message


        Examples:
        ./create-transaction.sh --api-key \\
            -d 101 

        ./create-transaction.sh --api-key \\
            -d 101 \\
            -m paypal \\
            -t payment

        ./create-transaction.sh --jwt \\
            -d 101 \\
            -m credit 
        
        ./create-transaction.sh --jwt \\
            -d 101 \\
            -m credit \\
            -t payment \\
            -s tentative \\
            -c 1200
            
EOF
exit 0
}


X_API_KEY="put_secure_random_string_here_for_api_token"
JWT_TOKEN="eyJhbGciOiJSUzUxMiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwiZ2xvYmFsIjp7Im5hbWUiOiJKb2huIERvZSIsInJvbGVzIjpbXX0sImlhdCI6MTUxNjIzOTAyMn0.ove6_7BWQRe9HQyphwDdbiaAchgn9ynC4-2EYEXFeVTDADC4P3XYv5uLisYg4Mx8BZOnkWX-5L82pFO1mUZM147gLKMsYlc-iMKXy4sKZPzhQ_XKnBR-EBIf5x_ZD1wpva9ti7Yrvd0vDi8YSFdqqf7R4RA11hv9kg-_gg1uea6sK-Q_eEqoet7ocqGVLu-ghhkZdVLxu9tWJFPNueILWv8vW1Y_u9fDtfOhw7Ugf5ysI9RXiO-tXEHKN2HnFPCkwccnMFt4PJRzU1VoOldz0xzzZRb-j2tlbjLqcQkjMwLEoPQpC4Wbl8DgkaVdTi2aNyH7EbWMynlSOZIYK0AFvQ"
DEBITOR_ID=1
METHOD="credit"
TYPE="due"
TRANSACTION_ID=""
AUTH_HEADER=""

while [[ $# -gt 0 ]]; do
    case $1 in
        -h|--help)
            _usage
            ;;
        -a|--api-key)
        if [[ -n "${AUTH_HEADER}" ]]; then
            echo "oh"
            _usage
        fi
        AUTH_HEADER="X-Api-Key: ${X_API_KEY}"
        shift
        ;;
        -j|--jwt)
        if [[ -n "${AUTH_HEADER}" ]]; then
            _usage
        fi
        AUTH_HEADER="Authorization: Bearer ${JWT_TOKEN}"
        shift
        ;;
        -d|--debitor)
        DEBITOR_ID=$2
        shift 2
        ;;
        -i|--transaction)
        TRANSACTION_ID=$2
        shift 2
        ;;
        -m|--method)
        METHOD=$2
        shift 2
        ;;
        -t|--type)
        TYPE=$2
        shift 2
        ;;
        -c|--cent)
        echo $2
        if ! [[ $2 =~ ^[0-9]+$ ]]; then
            _usage
        fi
        AMOUNT=$2
        shift 2
        ;;
        -s|--status)
        STATUS=$2
        shift 2
        ;;
        *)
        shift
        ;;

    esac
done

echo "creating transaction...."

CREATE_TRANSACTION_URL="http://localhost:9092/api/rest/v1/transactions"
JSON_DATA=$(cat <<EOF
{
  "debitor_id": ${DEBITOR_ID}, 
  "transaction_type": "${TYPE}",
  "transaction_identifier": "${TRANSACTION_ID}",
  "method": "${METHOD}",
  "amount": {
    "currency": "EUR",
    "gross_cent": ${AMOUNT:-14000},
    "vat_rate": 19.0
  },
  "comment": "This is a test transaction",
  "status": "${STATUS:-valid}",
  "payment_processor_information": {
    "id": 72168763,
    "booking_code": "something"
  },
  "payment_start_url": "none",
  "effective_date": "2023-01-20"
}
EOF
)

curl -X POST \
    --header "Content-Type: application/json" \
    --header "${AUTH_HEADER}" \
    --data "${JSON_DATA}" \
    --url "${CREATE_TRANSACTION_URL}" \
    -v
