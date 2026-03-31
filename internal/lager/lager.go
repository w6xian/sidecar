package lager

import (
	"context"
	"encoding/json"
	"fmt"
	"maps"
	"net/http"
	"strings"
	"time"

	"github.com/w6xian/sidecar/internal/muxhttp"
	"github.com/w6xian/sidecar/internal/utils"
	"github.com/w6xian/sidecar/internal/utils/array"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type contextKey string

func (k contextKey) String() string { return "net/api context value " + string(k) }

const (
	ContextKeyLogReq contextKey = "api_v_log"
	ContextKeyRegLog contextKey = "api_v_reg"
	ContextKeyLogMsg contextKey = "api_v_log_msg"
)

type LogReq struct {
	// 日志
	Lager *zap.Logger `json:"-"`

	LogId     int64  `db:"log_id" json:"log_id"`
	ShopId    int64  `db:"shop_id" json:"shop_id"`
	TraceId   int64  `db:"trace_id" json:"trace_id"`
	AppId     string `db:"app_id" json:"app_id"`
	SessionId int64  `db:"session_id" json:"session_id"`

	OperationTitle  string `db:"operation_title" json:"operation_title"`
	OperationDesc   string `db:"operation_desc" json:"operation_desc"`
	OperationType   string `db:"operation_type" json:"operation_type"`
	OperationModule string `db:"operation_module" json:"operation_module"`

	OperationParams string `db:"operation_params" json:"operation_params"`
	OperationResult string `db:"operation_result" json:"operation_result"`
	OperationStatus int64  `db:"operation_status" json:"operation_status"`

	Data map[string]*TablesChange `db:"data" json:"data"`

	ErrorMsg string `db:"error_msg" json:"error_msg"`

	// 本地日志无需记录
	// UserAgent  string `db:"user_agent" json:"user_agent"`
	// DeviceType string `db:"device_type" json:"device_type"`

	Duration    int64  `db:"duration" json:"duration"`
	HandlerId   int64  `db:"handler_id" json:"handler_id"`
	HandlerName string `db:"handler_name" json:"handler_name"`
}

func FromContext(ctx context.Context) *LogReq {
	logger, ok := ctx.Value(ContextKeyLogReq).(*LogReq)
	if !ok {
		logger = &LogReq{}
	}
	return logger
}

func RegLager(l *zap.Logger, info ...string) muxhttp.Decorator {
	return func(f muxhttp.Handler) muxhttp.Handler {
		return func(w http.ResponseWriter, r *http.Request) ([]byte, error) {
			ctx := context.WithValue(r.Context(), ContextKeyRegLog, l)
			ctx1 := context.WithValue(ctx, ContextKeyLogMsg, info)
			r = r.WithContext(ctx1)
			return f(w, r)
		}
	}
}

// Http Request 加 lager
func RequestLager(f muxhttp.Handler) muxhttp.Handler {
	return func(w http.ResponseWriter, r *http.Request) ([]byte, error) {
		s := time.Now()
		logReq := &LogReq{}
		// logReq.UserAgent = r.UserAgent()
		// logReq.DeviceType = r.Header.Get("X-Device-Type")
		logReq.OperationModule = fmt.Sprintf("%s %s", r.Method, r.URL.Path)
		logReq.OperationParams = r.URL.Query().Encode()

		if msg, ok := r.Context().Value(ContextKeyLogMsg).([]string); ok {
			if len(msg) > 0 {
				logReq.OperationTitle = msg[0]
			}
			if len(msg) > 1 {
				logReq.OperationDesc = msg[1]
			}
		}

		ctx := context.WithValue(r.Context(), ContextKeyLogReq, logReq)
		r = r.WithContext(ctx)
		rst, err := f(w, r)
		e := time.Now()
		if err != nil {
			logReq.SetErrorMsg(err.Error())
		} else {
			logReq.OperationStatus = 200
			logReq.OperationResult = string(rst)
		}
		logReq.Duration = e.UnixMilli() - s.UnixMilli()
		// 这里与decorate.go中的RequestId保持一致(int64)
		if traceId, ok := ctx.Value(muxhttp.ContextKey("request_id")).(int64); ok {
			logReq.TraceId = traceId
		}

		if array.InArray(r.Method, []string{http.MethodPost, http.MethodPut, http.MethodDelete, http.MethodPatch}) {
			if data, ok := ctx.Value(muxhttp.ContextKey("post_value")).(*muxhttp.HttpPostJson); ok {
				logReq.OperationParams = string(data.Data)
			}
		}

		if logger, ok := ctx.Value(ContextKeyRegLog).(*zap.Logger); ok {
			if err != nil {
				logReq.Error(r, logger)
			} else {
				logReq.Info(r, logger)
			}
		}
		return rst, err

	}
}

func (r *LogReq) SetOperation(title string, operationType string, operationModule string) {
	r.OperationTitle = title
	r.OperationType = operationType
	r.OperationModule = operationModule
}

func (r *LogReq) SetHandler(handlerId int64, handlerName string) {
	r.HandlerId = handlerId
	r.HandlerName = handlerName
}

func (r *LogReq) SetOperationParams(params string) {
	r.OperationParams = params
}

func (r *LogReq) SetDesc(desc string) {
	r.OperationDesc = desc
}

func (r *LogReq) SetParams(params string) {
	r.OperationParams = params
}

func (r *LogReq) SetResult(result string, status int64) {
	r.OperationResult = result
	r.OperationStatus = status
}

func (r *LogReq) SetData(data map[string]*TablesChange) {
	if r.Data == nil {
		r.Data = make(map[string]*TablesChange)
	}
	maps.Copy(r.Data, data)
}

func (r *LogReq) SetErrorMsg(msg string) {
	r.ErrorMsg = msg
}

type TablesChange struct {
	// 变更追踪
	TableChanges []*TableChange `json:"table_changes,omitempty"`
}

type TableChange struct {
	// 变更追踪
	BeforeData    string `json:"before_data,omitempty"`    // 变更前数据(JSON格式)
	AfterData     string `json:"after_data,omitempty"`     // 变更后数据(JSON格式)
	ChangedFields string `json:"changed_fields,omitempty"` // 变更字段列表(逗号分隔)
}

type Changeds struct {
	TableChanges []*Changed `json:"table_changes"`
}

func (r *Changeds) MarshalLogArray(enc zapcore.ArrayEncoder) error {
	for _, change := range r.TableChanges {
		enc.AppendObject(change)
	}
	return nil
}

type Changed struct {
	BeforeData    string `json:"before_data"`
	AfterData     string `json:"after_data"`
	ChangedFields string `json:"changed_fields"`
}

func (r *Changed) MarshalLogObject(enc zapcore.ObjectEncoder) error {
	enc.AddString("before_data", r.BeforeData)
	enc.AddString("after_data", r.AfterData)
	enc.AddString("changed_fields", r.ChangedFields)
	return nil
}

func CompareTableChange(t1, t2 map[string]any) *TableChange {
	tc := &TableChange{}

	// 对比t1与t2,去除值相同的字段
	if t1 == nil {
		t1 = make(map[string]any)
	}
	if t2 == nil {
		t2 = make(map[string]any)
	}
	b, _ := json.Marshal(t1)
	tc.BeforeData = string(b)
	c, _ := json.Marshal(t2)
	tc.AfterData = string(c)
	// 1. 移除t1中与t2值相同的字段
	for key, val1 := range t1 {
		if val2, exists := t2[key]; exists {
			// 只需要简单的对比,在数据库中,只存在string类型
			if utils.GetString(val1) == utils.GetString(val2) {
				delete(t1, key)
				delete(t2, key)
			}
		}
	}
	// 2. 添加t2中存在但t1中不存在的字段
	for key, val2 := range t2 {
		if _, exists := t1[key]; !exists {
			t1[key] = val2
		}
	}

	ks := []string{}
	for k := range t1 {
		ks = append(ks, k)
	}
	cp := fmt.Sprintf("[%s]", strings.Join(ks, ","))
	tc.ChangedFields = cp
	return tc
}

func (r *LogReq) buildFields(req *http.Request, logger *zap.Logger) []zap.Field {
	// 转换 data 为结构化日志字段
	var dataFields []zap.Field
	for tableName, change := range r.Data {
		tables := []*Changed{}
		for _, v := range change.TableChanges {
			tables = append(tables, &Changed{
				BeforeData:    v.BeforeData,
				AfterData:     v.AfterData,
				ChangedFields: v.ChangedFields,
			})
		}
		// 为每个表创建独立的Namespace
		dataFields = append(dataFields, zap.Array(tableName, &Changeds{
			TableChanges: tables,
		}))

	}
	//  把lager.proto转成 fields格式
	fields := []zap.Field{
		zap.Int64("shop_id", r.ShopId),
		zap.Int64("session_id", r.SessionId),
		zap.Int64("trace_id", r.TraceId),
		zap.String("app_id", r.AppId),
		zap.String("operation_type", r.OperationType),
		zap.String("operation_module", r.OperationModule),
		zap.String("operation_params", r.OperationParams),
		zap.String("operation_result", r.OperationResult), // 保留原字段
		zap.Int64("operation_status", r.OperationStatus),
		zap.Dict("data", dataFields...),
		zap.String("error_msg", r.ErrorMsg),
		// zap.String("user_agent", r.UserAgent),
		// zap.String("device_type", r.DeviceType),
		zap.Int64("duration", r.Duration),
		zap.Int64("handler_id", r.HandlerId),
		zap.String("handler_name", r.HandlerName),
	}

	return fields
}

func (r *LogReq) Debug(req *http.Request, logger *zap.Logger) {
	fields := r.buildFields(req, logger)
	logger.Debug(r.OperationTitle, fields...)
	logger.Sync()
}

func (r *LogReq) Info(req *http.Request, logger *zap.Logger) {
	fields := r.buildFields(req, logger)
	logger.Info(r.OperationTitle, fields...)
	logger.Sync()
}
func (r *LogReq) Error(req *http.Request, logger *zap.Logger) {
	fields := r.buildFields(req, logger)
	logger.Error(r.OperationTitle, fields...)
	logger.Sync()
}
