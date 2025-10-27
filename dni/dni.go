package dni

import (
	"errors"
	"regexp"
	"strconv"
	"strings"
)

var (
	ErrInvalidFormat = errors.New("invalid DNI/NIE format")
	ErrInvalidLetter = errors.New("invalid DNI/NIE control letter")
)

// TODO Replace to an external library

type DNI string

// GetLetter calcula la letra de control correspondiente al número del DNI o NIE
func (d DNI) GetLetter() string {
	dniStr := strings.ToUpper(string(d))

	// Si es NIE, convertir la letra inicial a número
	if d.IsNIE() {
		replacement := map[rune]rune{'X': '0', 'Y': '1', 'Z': '2'}
		firstChar := rune(dniStr[0])
		if newChar, ok := replacement[firstChar]; ok {
			dniStr = string(newChar) + dniStr[1:]
		}
	}

	// Expresión regular para extraer los 8 dígitos
	regex := regexp.MustCompile(`^[0-9]{8}`)

	if !regex.MatchString(dniStr) {
		return ""
	}

	// Extraer los primeros 8 dígitos
	numero := dniStr[:8]
	num, err := strconv.Atoi(numero)
	if err != nil {
		return ""
	}

	// Tabla de letras válidas según el resto de dividir el número entre 23
	letrasValidas := "TRWAGMYFPDXBNJZSQVHLCKE"
	return string(letrasValidas[num%23])
}

func (d DNI) IsValid() error {
	return d.validateWithNIE()
}

// IsNIE verifica si es un NIE (Número de Identidad de Extranjero) en lugar de DNI
// Los NIE empiezan con X, Y o Z seguido de 7 dígitos y una letra
func (d DNI) IsNIE() bool {
	regex := regexp.MustCompile(`^[XYZ][0-9]{7}[A-Za-z]$`)
	return regex.MatchString(strings.ToUpper(string(d)))
}

// validateWithNIE valida tanto DNI como NIE españoles (método privado)
func (d DNI) validateWithNIE() error {
	dniStr := strings.ToUpper(string(d))

	// Validar formato: DNI (8 dígitos + letra) o NIE (X/Y/Z + 7 dígitos + letra)
	regexDNI := regexp.MustCompile(`^[0-9]{8}[A-Za-z]$`)
	regexNIE := regexp.MustCompile(`^[XYZ][0-9]{7}[A-Za-z]$`)

	if !regexDNI.MatchString(dniStr) && !regexNIE.MatchString(dniStr) {
		return ErrInvalidFormat
	}

	// Extraer la letra proporcionada
	letraProporcionada := string(dniStr[len(dniStr)-1])

	// Obtener la letra esperada
	letraEsperada := d.GetLetter()

	// Comparar las letras
	if letraEsperada != letraProporcionada {
		return ErrInvalidLetter
	}
	return nil
}

// Normalize normaliza el DNI a formato estándar (mayúsculas, sin espacios ni guiones)
func (d DNI) Normalize() DNI {
	dniStr := string(d)
	// Eliminar espacios, guiones y otros caracteres especiales
	dniStr = strings.ReplaceAll(dniStr, " ", "")
	dniStr = strings.ReplaceAll(dniStr, "-", "")
	dniStr = strings.ToUpper(dniStr)
	return DNI(dniStr)
}

// GetNumber devuelve solo la parte numérica del DNI (los 8 dígitos o 7 en caso de NIE)
func (d DNI) GetNumber() string {
	dniStr := string(d)

	if d.IsNIE() {
		regex := regexp.MustCompile(`^[XYZ]([0-9]{7})`)
		matches := regex.FindStringSubmatch(dniStr)
		if len(matches) > 1 {
			return matches[1]
		}
		return ""
	}

	regex := regexp.MustCompile(`^[0-9]{8}`)
	if regex.MatchString(dniStr) {
		return dniStr[:8]
	}
	return ""
}

// Mask enmascara el DNI para mostrar solo los últimos 3 dígitos y la letra
// Útil para logs, interfaces de usuario, cumplimiento GDPR/LOPD
// Ejemplo: "12345678Z" -> "*****678Z"
// Ejemplo NIE: "X1234567L" -> "X****67L"
func (d DNI) Mask() string {
	if d.IsValid() != nil {
		return "***INVALID***"
	}

	dniStr := strings.ToUpper(string(d))

	// Para NIE: mostrar X/Y/Z + asteriscos + últimos 2 dígitos + letra
	if d.IsNIE() {
		return string(dniStr[0]) + "****" + dniStr[6:]
	}

	// Para DNI: asteriscos + últimos 3 dígitos + letra
	return "*****" + dniStr[5:]
}

// Compare compara dos DNIs normalizándolos primero
// Útil para búsquedas y deduplicación
func (d DNI) Compare(other DNI) bool {
	return d.Normalize() == other.Normalize()
}
