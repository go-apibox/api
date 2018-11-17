// 错误定义

package api

// global error
const (
	errorActionNotExist = iota
	errorSystemMaintenance
)

var globalErrorDefines = map[ErrorType]*ErrorDefine{
	errorActionNotExist: &ErrorDefine{
		code:        "ActionNotExist",
		fieldCounts: []int{0},
		msgTmpls: map[string]map[int]string{
			"en_us": {
				0: "API action does not exist!",
			},
			"zh_cn": {
				0: "接口不存在！",
			},
		},
	},
	errorSystemMaintenance: &ErrorDefine{
		code:        "SystemMaintenance",
		fieldCounts: []int{0},
		msgTmpls: map[string]map[int]string{
			"en_us": {
				0: "System is under maintenance!",
			},
			"zh_cn": {
				0: "系统维护中！",
			},
		},
	},
}

// application error
const (
	ErrorObjectNotExist = iota
	ErrorObjectDuplicated
	ErrorNoObjectUpdated
	ErrorNoObjectDeleted
	ErrorMissingParam
	ErrorInvalidParam
	ErrorQuotaExceed
	ErrorPermissionDenied
	ErrorOperationFailed
	ErrorInternalError
)

var appErrorDefines = map[ErrorType]*ErrorDefine{
	ErrorObjectNotExist: &ErrorDefine{
		code:        "ObjectNotExist",
		fieldCounts: []int{0, 1},
		msgTmpls: map[string]map[int]string{
			"en_us": {
				0: "Object does not exist!",
				1: "{1} does not exist!",
			},
			"zh_cn": {
				0: "对象不存在！",
				1: "{1}不存在！",
			},
		},
	},
	ErrorObjectDuplicated: &ErrorDefine{
		code:        "ObjectDuplicated",
		fieldCounts: []int{0, 1, 2},
		msgTmpls: map[string]map[int]string{
			"en_us": {
				0: "Object duplicated!",
				1: "{1} already exists!",
				2: "{1} with same {2} already exists!",
			},
			"zh_cn": {
				0: "对象已存在！",
				1: "{1}已存在！",
				2: "相同{2}的{1}已存在！",
			},
		},
	},
	ErrorNoObjectUpdated: &ErrorDefine{
		code:        "NoObjectUpdated",
		fieldCounts: []int{0, 1},
		msgTmpls: map[string]map[int]string{
			"en_us": {
				0: "No object updated!",
				1: "No {1} updated!",
			},
			"zh_cn": {
				0: "没有对象被更新！",
				1: "没有{1}被更新！",
			},
		},
	},
	ErrorNoObjectDeleted: &ErrorDefine{
		code:        "NoObjectDeleted",
		fieldCounts: []int{0, 1},
		msgTmpls: map[string]map[int]string{
			"en_us": {
				0: "No object deleted!",
				1: "No {1} deleted!",
			},
			"zh_cn": {
				0: "没有对象被删除！",
				1: "没有{1}被删除！",
			},
		},
	},
	ErrorMissingParam: &ErrorDefine{
		code:        "MissingParam",
		fieldCounts: []int{1},
		msgTmpls: map[string]map[int]string{
			"en_us": {
				1: "Missing param {1: or }!",
			},
			"zh_cn": {
				1: "缺少{1:或}！",
			},
		},
	},
	ErrorInvalidParam: &ErrorDefine{
		code:        "InvalidParam",
		fieldCounts: []int{1, 2},
		msgTmpls: map[string]map[int]string{
			"en_us": {
				1: "Invalid param {1: or }!",
				2: "Invalid param {1: or }: {2}!",
			},
			"zh_cn": {
				1: "无效的{1:或}！",
				2: "无效的{1:或}：{2}！",
			},
		},
	},
	ErrorQuotaExceed: &ErrorDefine{
		code:        "QuotaExceed",
		fieldCounts: []int{0, 1},
		msgTmpls: map[string]map[int]string{
			"en_us": {
				0: "Quota exceed!",
				1: "Quota exceed: {1}!",
			},
			"zh_cn": {
				0: "超出配额限制！",
				1: "超出配额限制：{1}！",
			},
		},
	},
	ErrorPermissionDenied: &ErrorDefine{
		code:        "PermissionDenied",
		fieldCounts: []int{0, 1},
		msgTmpls: map[string]map[int]string{
			"en_us": {
				0: "Permission denied!",
				1: "Permission denied: {1}!",
			},
			"zh_cn": {
				0: "没有权限！",
				1: "没有权限：{1}！",
			},
		},
	},
	ErrorOperationFailed: &ErrorDefine{
		code:        "OperationFailed",
		fieldCounts: []int{1},
		msgTmpls: map[string]map[int]string{
			"en_us": {
				1: "Operation failed: {1}!",
			},
			"zh_cn": {
				1: "操作失败：{1}！",
			},
		},
	},
	ErrorInternalError: &ErrorDefine{
		code:        "InternalError",
		fieldCounts: []int{1, 2},
		msgTmpls: map[string]map[int]string{
			"en_us": {
				1: "Internal error: {1}!",
				2: "Internal error: {1} {2}!",
			},
			"zh_cn": {
				1: "服务器内部错误：{1}！",
				2: "服务器内部错误：{1}{2}！",
			},
		},
	},
}
