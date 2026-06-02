// Package validate содержит функции проверки входных данных.
package validate

import "unicode"

// OrderNumber проверяет номер заказа: только цифры и контрольная сумма по алгоритму Луна.
func OrderNumber(number string) bool {
	if number == "" {
		return false
	}

	for _, r := range number {
		if !unicode.IsDigit(r) {
			return false
		}
	}

	sum := 0
	alternate := false
	for i := len(number) - 1; i >= 0; i-- {
		d := int(number[i] - '0')
		if alternate {
			d *= 2
			if d > 9 {
				d -= 9
			}
		}
		sum += d
		alternate = !alternate
	}

	return sum%10 == 0
}
