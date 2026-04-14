<template>
  <div class="page-content analytics-page">
    <AppToast ref="toast" />

    <!-- Page header -->
    <div class="page-header">
      <div>
        <h2 class="page-header__title">Analytics</h2>
        <AppBreadcrumb :items="contextStore.contextBreadcrumb" @navigate="onBreadcrumbNav" />
      </div>
      <div class="page-header__actions">
        <label class="auto-refresh-toggle">
          <input type="checkbox" v-model="autoRefresh" />
          <span class="text-sm">Auto-refresh</span>
        </label>
        <AppButton variant="outline" size="sm" :loading="refreshing" @click="refreshAll">
          <svg width="14" height="14" viewBox="0 0 14 14" fill="none"><path d="M1.5 7A5.5 5.5 0 0112 4M12.5 7A5.5 5.5 0 012 10" stroke="currentColor" stroke-width="1.5" stroke-linecap="round"/></svg>
          Refresh
        </AppButton>
      </div>
    </div>

    <!-- Scope filter chips -->
    <div class="analytics-page__filters">
      <div class="filter-chips">
        <button
          v-for="chip in scopeChips"
          :key="chip.id"
          class="scope-chip"
          :class="{ 'scope-chip--active': activeScope === chip.id }"
          @click="activeScope = chip.id"
        >{{ chip.label }}</button>
      </div>

      <!-- Date range -->
      <div class="date-range">
        <button
          v-for="preset in datePresets"
          :key="preset.value"
          class="date-btn"
          :class="{ 'date-btn--active': dateRange === preset.value }"
          @click="dateRange = preset.value"
        >{{ preset.label }}</button>
        <div v-if="dateRange === 'custom'" class="date-custom">
          <AppInput v-model="customStart" type="date" placeholder="Start" />
          <span class="text-muted">to</span>
          <AppInput v-model="customEnd" type="date" placeholder="End" />
        </div>
      </div>
    </div>

    <!-- KPI tiles -->
    <div class="kpi-grid">
      <template v-if="kpiLoading">
        <div v-for="n in 6" :key="n" class="kpi-tile kpi-tile--skeleton">
          <div class="skeleton" style="width: 40px; height: 40px; border-radius: 8px" />
          <div>
            <div class="skeleton" style="width: 80px; height: 12px; margin-bottom: 8px" />
            <div class="skeleton" style="width: 60px; height: 24px; margin-bottom: 4px" />
            <div class="skeleton" style="width: 50px; height: 12px" />
          </div>
        </div>
      </template>
      <template v-else-if="kpis.length > 0">
        <div v-for="kpi in kpis" :key="kpi.key" class="kpi-tile">
          <div class="kpi-tile__icon" :class="`kpi-tile__icon--${kpi.color || 'primary'}`">
            <svg width="22" height="22" viewBox="0 0 22 22" fill="none">
              <path v-if="kpi.icon === 'users'" d="M16 18v-1a4 4 0 00-4-4H8a4 4 0 00-4 4v1M10 9a3 3 0 100-6 3 3 0 000 6zM18 9a2 2 0 100-4 2 2 0 000 4z" stroke="currentColor" stroke-width="1.5" stroke-linecap="round"/>
              <path v-else-if="kpi.icon === 'chart'" d="M3 19h18M5 15v4M9 11v8M13 7v12M17 3v16" stroke="currentColor" stroke-width="1.5" stroke-linecap="round"/>
              <path v-else-if="kpi.icon === 'clock'" d="M11 3a8 8 0 100 16 8 8 0 000-16zM11 6v5l3 3" stroke="currentColor" stroke-width="1.5" stroke-linecap="round"/>
              <path v-else d="M3 11l4 4L19 3" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"/>
            </svg>
          </div>
          <div class="kpi-tile__body">
            <span class="kpi-tile__label">{{ kpi.label }}</span>
            <span class="kpi-tile__value">{{ kpi.formattedValue || kpi.value }}</span>
            <span class="kpi-tile__trend" :class="kpi.changePercent >= 0 ? 'kpi-tile__trend--up' : 'kpi-tile__trend--down'">
              <svg v-if="kpi.changePercent >= 0" width="12" height="12" viewBox="0 0 12 12" fill="none"><path d="M6 9V3M3 5l3-3 3 3" stroke="currentColor" stroke-width="1.5" stroke-linecap="round"/></svg>
              <svg v-else width="12" height="12" viewBox="0 0 12 12" fill="none"><path d="M6 3v6M3 7l3 3 3-3" stroke="currentColor" stroke-width="1.5" stroke-linecap="round"/></svg>
              {{ Math.abs(kpi.changePercent || 0).toFixed(1) }}%
            </span>
          </div>
        </div>
      </template>
      <AppEmptyState v-else title="No KPIs available" description="No data for the selected scope and date range." />
    </div>

    <!-- Error state for KPIs -->
    <AppErrorState v-if="kpiError" :message="kpiError" retryable @retry="loadKPIs" />

    <!-- Charts section -->
    <div class="charts-section">
      <div class="charts-section__toolbar">
        <h3>Trends</h3>
        <div class="charts-section__actions">
          <AppButton variant="ghost" size="sm" @click="resetChartZoom">Reset Zoom</AppButton>
          <AppButton variant="ghost" size="sm" @click="downloadChart">
            <svg width="14" height="14" viewBox="0 0 14 14" fill="none"><path d="M7 2v8M4 7l3 3 3-3M2 12h10" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"/></svg>
            Export
          </AppButton>
        </div>
      </div>

      <AppLoadingState v-if="trendsLoading" message="Loading charts..." />
      <AppErrorState v-else-if="trendsError" :message="trendsError" retryable @retry="loadTrends" />

      <div v-else-if="trendData" class="charts-grid">
        <div class="chart-card">
          <h4 class="chart-card__title">Time Series</h4>
          <div ref="lineChartRef" class="chart-card__canvas" />
        </div>
        <div class="chart-card">
          <h4 class="chart-card__title">Category Comparison</h4>
          <div ref="barChartRef" class="chart-card__canvas" />
        </div>
      </div>

      <AppEmptyState v-else title="No trend data" description="Select a different scope or date range to view trends." />
    </div>
  </div>
</template>

<script setup>
import { ref, computed, watch, onMounted, onBeforeUnmount, nextTick } from 'vue';
import { useContextStore } from '@/stores/context.js';
import * as analyticsApi from '@/api/analytics.js';
import AppButton from '@/components/common/AppButton.vue';
import AppInput from '@/components/common/AppInput.vue';
import AppBreadcrumb from '@/components/common/AppBreadcrumb.vue';
import AppLoadingState from '@/components/common/AppLoadingState.vue';
import AppErrorState from '@/components/common/AppErrorState.vue';
import AppEmptyState from '@/components/common/AppEmptyState.vue';
import AppToast from '@/components/common/AppToast.vue';

const contextStore = useContextStore();
const toast = ref(null);

// ---- Filters ----
const activeScope = ref('all');
const dateRange = ref('30d');
const customStart = ref('');
const customEnd = ref('');
const autoRefresh = ref(false);
let refreshInterval = null;

const scopeChips = computed(() => {
  const chips = [{ id: 'all', label: 'All' }];
  if (contextStore.currentNode) {
    chips.push({ id: contextStore.currentNode.id, label: contextStore.currentNode.name || 'Current Node' });
  }
  return chips;
});

const datePresets = [
  { value: 'today', label: 'Today' },
  { value: '7d', label: '7 days' },
  { value: '30d', label: '30 days' },
  { value: '90d', label: '90 days' },
  { value: 'custom', label: 'Custom' },
];

function buildParams() {
  const params = { range: dateRange.value };
  if (activeScope.value !== 'all') params.nodeId = activeScope.value;
  if (dateRange.value === 'custom') {
    params.startDate = customStart.value;
    params.endDate = customEnd.value;
  }
  return params;
}

// ---- KPIs ----
const kpis = ref([]);
const kpiLoading = ref(false);
const kpiError = ref('');
const refreshing = ref(false);

async function loadKPIs() {
  kpiLoading.value = true;
  kpiError.value = '';
  try {
    const { data } = await analyticsApi.getKPIs(buildParams());
    kpis.value = Array.isArray(data) ? data : data.items || [];
  } catch (err) {
    kpiError.value = err.message || 'Failed to load KPIs';
  } finally {
    kpiLoading.value = false;
  }
}

// ---- Trends / Charts ----
const trendData = ref(null);
const trendsLoading = ref(false);
const trendsError = ref('');
const lineChartRef = ref(null);
const barChartRef = ref(null);
let lineChart = null;
let barChart = null;
let echarts = null;

async function loadTrends() {
  trendsLoading.value = true;
  trendsError.value = '';
  try {
    const { data } = await analyticsApi.getTrends(buildParams());
    trendData.value = data;
    await nextTick();
    renderCharts(data);
  } catch (err) {
    trendsError.value = err.message || 'Failed to load trends';
  } finally {
    trendsLoading.value = false;
  }
}

async function ensureEcharts() {
  if (!echarts) {
    try {
      echarts = await import('echarts');
    } catch {
      trendsError.value = 'Chart library not available.';
      return false;
    }
  }
  return true;
}

async function renderCharts(data) {
  if (!(await ensureEcharts())) return;

  const lineLabels = data.timeSeries?.labels || data.labels || [];
  const lineValues = data.timeSeries?.values || data.values || [];
  const barCategories = data.categories?.labels || data.barLabels || [];
  const barValues = data.categories?.values || data.barValues || [];

  // Line chart
  if (lineChartRef.value) {
    if (lineChart) lineChart.dispose();
    lineChart = echarts.init(lineChartRef.value);
    lineChart.setOption({
      tooltip: { trigger: 'axis' },
      xAxis: { type: 'category', data: lineLabels, boundaryGap: false },
      yAxis: { type: 'value' },
      dataZoom: [{ type: 'inside' }],
      series: [{ type: 'line', data: lineValues, smooth: true, areaStyle: { opacity: 0.15 }, itemStyle: { color: '#1a73e8' } }],
      grid: { left: 48, right: 16, top: 16, bottom: 32 },
    });
  }

  // Bar chart
  if (barChartRef.value) {
    if (barChart) barChart.dispose();
    barChart = echarts.init(barChartRef.value);
    barChart.setOption({
      tooltip: { trigger: 'axis' },
      xAxis: { type: 'category', data: barCategories },
      yAxis: { type: 'value' },
      series: [{ type: 'bar', data: barValues, itemStyle: { color: '#34a853', borderRadius: [4, 4, 0, 0] } }],
      grid: { left: 48, right: 16, top: 16, bottom: 32 },
    });
  }
}

function resetChartZoom() {
  lineChart?.dispatchAction({ type: 'dataZoom', start: 0, end: 100 });
  barChart?.dispatchAction({ type: 'dataZoom', start: 0, end: 100 });
}

function downloadChart() {
  if (!lineChart) return;
  const url = lineChart.getDataURL({ type: 'png', pixelRatio: 2, backgroundColor: '#fff' });
  const a = document.createElement('a');
  a.href = url;
  a.download = 'analytics-chart.png';
  a.click();
}

function handleResize() {
  lineChart?.resize();
  barChart?.resize();
}

// ---- Refresh ----
async function refreshAll() {
  refreshing.value = true;
  await Promise.all([loadKPIs(), loadTrends()]);
  refreshing.value = false;
}

watch(autoRefresh, (val) => {
  clearInterval(refreshInterval);
  if (val) {
    refreshInterval = setInterval(refreshAll, 60000);
  }
});

watch([activeScope, dateRange, customStart, customEnd], () => {
  refreshAll();
});

// Context change
function onBreadcrumbNav(item) {
  contextStore.switchContext(item.id);
}
window.addEventListener('context:changed', refreshAll);

onMounted(() => {
  refreshAll();
  window.addEventListener('resize', handleResize);
});

onBeforeUnmount(() => {
  clearInterval(refreshInterval);
  window.removeEventListener('context:changed', refreshAll);
  window.removeEventListener('resize', handleResize);
  lineChart?.dispose();
  barChart?.dispose();
});
</script>

<style lang="scss" scoped>
.analytics-page {
  &__filters {
    display: flex;
    align-items: center;
    justify-content: space-between;
    flex-wrap: wrap;
    gap: $space-4;
    margin-bottom: $space-6;
  }
}

.filter-chips {
  display: flex;
  gap: $space-2;
}

.scope-chip {
  padding: $space-1 $space-4;
  border-radius: $border-radius-full;
  font-size: $font-size-sm;
  font-weight: $font-weight-medium;
  border: 1px solid $border-color;
  background: $color-neutral-0;
  color: $color-neutral-600;
  cursor: pointer;
  transition: all $transition-fast;

  &:hover { border-color: $color-primary-300; color: $color-primary-600; }
  &--active {
    background: $color-primary-500;
    border-color: $color-primary-500;
    color: #fff;
  }
}

.date-range {
  display: flex;
  align-items: center;
  gap: $space-2;
  flex-wrap: wrap;
}

.date-btn {
  padding: $space-1 $space-3;
  border-radius: $border-radius-base;
  font-size: $font-size-sm;
  font-weight: $font-weight-medium;
  border: 1px solid $border-color;
  background: $color-neutral-0;
  color: $color-neutral-600;
  cursor: pointer;
  transition: all $transition-fast;

  &:hover { border-color: $color-primary-300; }
  &--active {
    background: $color-primary-50;
    border-color: $color-primary-500;
    color: $color-primary-700;
  }
}

.date-custom {
  display: flex;
  align-items: center;
  gap: $space-2;
}

.auto-refresh-toggle {
  display: flex;
  align-items: center;
  gap: $space-2;
  cursor: pointer;
  color: $color-neutral-600;
  input { accent-color: $color-primary-500; }
}

// KPI tiles
.kpi-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(220px, 1fr));
  gap: $space-4;
  margin-bottom: $space-7;
}

.kpi-tile {
  display: flex;
  align-items: flex-start;
  gap: $space-4;
  padding: $space-5;
  background: $color-neutral-0;
  border: 1px solid $border-color;
  border-radius: $border-radius-md;
  box-shadow: $shadow-xs;
  transition: box-shadow $transition-fast;

  &:hover { box-shadow: $shadow-sm; }

  &--skeleton {
    animation: pulse-bg 1.5s ease-in-out infinite;
  }

  &__icon {
    width: 44px;
    height: 44px;
    border-radius: $border-radius-md;
    display: flex;
    align-items: center;
    justify-content: center;
    flex-shrink: 0;

    &--primary { background: $color-primary-50; color: $color-primary-600; }
    &--success { background: $color-success-50; color: $color-success-600; }
    &--warning { background: $color-warning-50; color: $color-warning-600; }
    &--danger { background: $color-danger-50; color: $color-danger-600; }
  }

  &__body {
    display: flex;
    flex-direction: column;
    min-width: 0;
  }

  &__label {
    font-size: $font-size-sm;
    color: $color-neutral-500;
    margin-bottom: $space-1;
  }

  &__value {
    font-size: $font-size-2xl;
    font-weight: $font-weight-bold;
    color: $color-neutral-900;
    line-height: $line-height-tight;
  }

  &__trend {
    display: inline-flex;
    align-items: center;
    gap: 2px;
    font-size: $font-size-xs;
    font-weight: $font-weight-medium;
    margin-top: $space-1;

    &--up { color: $color-success-600; }
    &--down { color: $color-danger-600; }
  }
}

@keyframes pulse-bg {
  0%, 100% { opacity: 1; }
  50% { opacity: 0.6; }
}

// Charts
.charts-section {
  &__toolbar {
    display: flex;
    align-items: center;
    justify-content: space-between;
    margin-bottom: $space-4;
    h3 { font-size: $font-size-xl; color: $color-neutral-800; margin: 0; }
  }
  &__actions {
    display: flex;
    gap: $space-2;
  }
}

.charts-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(400px, 1fr));
  gap: $space-4;
  @media (max-width: 900px) { grid-template-columns: 1fr; }
}

.chart-card {
  background: $color-neutral-0;
  border: 1px solid $border-color;
  border-radius: $border-radius-md;
  padding: $space-5;
  box-shadow: $shadow-xs;

  &__title {
    font-size: $font-size-base;
    font-weight: $font-weight-semibold;
    color: $color-neutral-700;
    margin: 0 0 $space-4;
  }

  &__canvas {
    width: 100%;
    height: 320px;
  }
}

.skeleton {
  background: linear-gradient(90deg, $color-neutral-100 25%, $color-neutral-50 50%, $color-neutral-100 75%);
  background-size: 200% 100%;
  animation: shimmer 1.5s infinite;
  border-radius: $border-radius-base;
}

@keyframes shimmer {
  0% { background-position: 200% 0; }
  100% { background-position: -200% 0; }
}
</style>
