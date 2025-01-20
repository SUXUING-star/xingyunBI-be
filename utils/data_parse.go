package utils

import (
	"encoding/json"
	"errors"
	"fmt"
	"mime/multipart"

	"github.com/xuri/excelize/v2"
)

// ParseExcelFile 解析Excel文件
func ParseExcelFile(file multipart.File) ([][]string, []string, error) {
	xlsx, err := excelize.OpenReader(file)
	if err != nil {
		return nil, nil, err
	}
	defer xlsx.Close()

	// 获取第一个工作表
	sheets := xlsx.GetSheetList()
	if len(sheets) == 0 {
		return nil, nil, errors.New("excel文件没有工作表")
	}

	// 获取所有行
	rows, err := xlsx.GetRows(sheets[0])
	if err != nil {
		return nil, nil, err
	}

	if len(rows) == 0 {
		return nil, nil, errors.New("excel文件为空")
	}

	// 第一行作为表头
	headers := rows[0]
	content := rows[1:]

	return content, headers, nil
}

// ParseJSONFile 解析JSON文件
func ParseJSONFile(file multipart.File) ([][]string, []string, error) {
	var jsonData []map[string]interface{}
	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&jsonData); err != nil {
		return nil, nil, err
	}

	if len(jsonData) == 0 {
		return nil, nil, errors.New("JSON数据为空")
	}

	// 提取表头
	headers := make([]string, 0)
	for key := range jsonData[0] {
		headers = append(headers, key)
	}

	// 将数据转换为二维字符串数组
	var content [][]string
	for _, item := range jsonData {
		var row []string
		for _, header := range headers {
			value := fmt.Sprint(item[header])
			row = append(row, value)
		}
		content = append(content, row)
	}

	return content, headers, nil
}
