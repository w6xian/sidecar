package ean

import (
	"errors"
	"fmt"
	"strconv"
)

// ValidateEAN13 验证EAN13条形码是否符合规则
// 返回nil表示验证通过，否则返回错误信息
func ValidateEAN13(code string) error {
	sum, err := SumEAN13(code)
	if err != nil {
		return err
	}
	// 比较计算的校验位与提供的校验位
	providedChecksum, _ := strconv.Atoi(string(code[12]))
	if sum != providedChecksum {
		return errors.New("EAN13 checksum is invalid")
	}

	return nil
}

func SumEAN13(code string) (int, error) {
	// 取前12位
	code = code[:12]

	// 检查是否只包含数字
	for _, c := range code {
		if c < '0' || c > '9' {
			return 0, errors.New("EAN13 code must contain only digits")
		}
	}

	// 计算校验位
	var sumEven, sumOdd int
	for i := range 12 {
		digit, _ := strconv.Atoi(string(code[i]))
		if i%2 == 0 {
			// 奇数位 (位置1,3,5,...11) - 从左数，第一位是位置1
			sumOdd += digit
		} else {
			// 偶数位 (位置2,4,6,...12)
			sumEven += digit
		}
	}

	// EAN13算法：奇数位和 + 偶数位和×3
	checksum := (sumOdd + sumEven*3) % 10
	if checksum != 0 {
		checksum = 10 - checksum
	}
	return checksum, nil
}

// GenerateStoreEAN13 生成店内EAN13条形码
// 规则：首位固定为2+6位商品ID+5位商品价格+1位校验码
// productID: 商品ID，将被格式化为6位数字
// price: 商品价格（单位：分），将被格式化为5位数字
// 返回生成的13位EAN13条形码字符串，以及可能的错误
func GenerateEAN13(productID int, price int) (string, error) {
	// 1. 验证商品ID是否为非负整数
	if productID < 0 {
		return "", fmt.Errorf("productID must be a non-negative integer")
	}
	if productID > 999999 {
		return "", fmt.Errorf("productID must not exceed 6 digits")
	}

	// 2. 格式化商品ID为6位，不足前面补0
	productIDFormatted := fmt.Sprintf("%06d", productID)
	if len(productIDFormatted) > 6 {
		return "", fmt.Errorf("productID must not exceed 6 digits")
	}

	// 3. 验证价格是否为正数且在5位数字范围内
	if price < 0 || price > 99999 {
		return "", fmt.Errorf("price must be between 0 and 99999")
	}

	// 4. 格式化价格为5位，不足前面补0
	priceFormatted := fmt.Sprintf("%05d", price)

	// 5. 构建前12位字符串（包含首位固定为2）
	first12Digits := "2" + productIDFormatted + priceFormatted

	// 6. 计算校验位
	checksum, err := SumEAN13(first12Digits)
	if err != nil {
		return "", fmt.Errorf("failed to calculate checksum: %w", err)
	}

	// 7. 组合完整的13位EAN13条形码
	ean13Code := first12Digits + strconv.Itoa(checksum)

	return ean13Code, nil
}

// ParseEAN13 解析店内EAN13条形码
// 规则：首位固定为2+6位商品ID+5位商品价格(分)+1位校验码
// 返回商品ID、价格(元)和可能的错误
func ParseEAN13(code string) (productID int, price int, err error) {
	// 1. 基础格式验证
	if len(code) != 13 {
		return 0, 0, errors.New("EAN13长度必须为13位")
	}
	if code[0] != '2' {
		return 0, 0, errors.New("非店内码EAN13，首位必须为2")
	}

	// 2. 校验码验证
	if err := ValidateEAN13(code); err != nil {
		return 0, 0, errors.New("校验码错误: " + err.Error())
	}

	// 3. 提取商品ID (第2-7位)
	productID, err = strconv.Atoi(code[1:7])
	if err != nil {
		return 0, 0, errors.New("商品ID格式错误: " + err.Error())
	}

	// 4. 提取价格 (第8-12位)，转换为元
	priceStr := code[7:12]
	priceInt, err := strconv.Atoi(priceStr)
	if err != nil {
		return 0, 0, errors.New("价格格式错误: " + err.Error())
	}
	price = priceInt
	return productID, price, nil
}
