package models

import "testing"

func TestTrendClusterTaskTableName(t *testing.T) {
	task := TrendClusterTask{}
	if got := task.TableName(); got != "trend_cluster_task" {
		t.Errorf("expected trend_cluster_task, got %s", got)
	}
}

func TestTrendOrzdbaCalcInstanceTableName(t *testing.T) {
	inst := TrendOrzdbaCalcInstance{}
	if got := inst.TableName(); got != "trend_orzdba_calc_instance" {
		t.Errorf("expected trend_orzdba_calc_instance, got %s", got)
	}
}

func TestTrendQuantileResultTableName(t *testing.T) {
	result := TrendQuantileResult{}
	if got := result.TableName(); got != "trend_quantile_result" {
		t.Errorf("expected trend_quantile_result, got %s", got)
	}
}
