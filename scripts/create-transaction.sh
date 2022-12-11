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
        -h, --help                  Show this message


        Examples:
        ./create-transaction.sh --api-key \\
            -d 101 

        ./create-transaction.sh --api-key \\
            -d 101 \\
            -m paypal \\
            -t payment
EOF
exit 0
}


X_API_KEY="put_secure_random_string_here_for_api_token"
JWT_TOKEN="TODO"
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
        *)
        shift
        ;;

    esac
done


CREATE_TRANSACTION_URL="http://localhost:9092/api/rest/v1/transactions"
JSON_DATA=$(cat <<EOF
{
  "debitor_id": ${DEBITOR_ID}, 
  "transaction_type": "${TYPE}",
  "transaction_identifier": "${TRANSACTION_ID}",
  "method": "${METHOD}",
  "amount": {
    "currency": "EUR",
    "gross_cent": 14000,
    "vat_rate": 7.7
  },
  "comment": "This is a test transaction",
  "status": "valid",
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
