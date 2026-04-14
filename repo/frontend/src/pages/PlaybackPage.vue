<template>
  <div class="playback-page" :style="pageBackground">
    <AppToast ref="toast" />

    <div class="playback-page__top">
      <!-- Collapsible sidebar -->
      <Transition name="slide-right">
        <aside v-if="sidebarOpen" class="playback-page__sidebar">
          <div class="sidebar__header">
            <h4>Media Library</h4>
            <button class="sidebar__close" @click="sidebarOpen = false" aria-label="Close sidebar">
              <svg width="18" height="18" viewBox="0 0 18 18" fill="none">
                <path d="M13.5 4.5L4.5 13.5M4.5 4.5l9 9" stroke="currentColor" stroke-width="1.5" stroke-linecap="round"/>
              </svg>
            </button>
          </div>
          <div class="sidebar__list">
            <div
              v-for="track in mediaList"
              :key="track.id"
              class="sidebar__track"
              :class="{ 'sidebar__track--active': currentTrack?.id === track.id }"
              @click="selectTrack(track)"
            >
              <div class="sidebar__track-art">
                <img v-if="track.coverUrl" :src="track.coverUrl" :alt="track.title" />
                <div v-else class="sidebar__track-art-placeholder">
                  <svg width="20" height="20" viewBox="0 0 20 20" fill="none"><circle cx="10" cy="10" r="8" stroke="currentColor" stroke-width="1.5"/><path d="M8 13V7l5 3-5 3z" fill="currentColor"/></svg>
                </div>
              </div>
              <div class="sidebar__track-info">
                <span class="sidebar__track-title">{{ track.title }}</span>
                <span class="sidebar__track-artist">{{ track.artist }}</span>
              </div>
            </div>
            <AppEmptyState v-if="!mediaLoading && mediaList.length === 0" title="No tracks" description="Upload media to get started." />
          </div>
          <div class="sidebar__upload">
            <AppFileUpload
              accept=".mp3,.wav,.flac,.m4a,.ogg"
              hint="Audio: mp3, wav, flac, m4a"
              @file-selected="onMediaFileSelected"
            />
            <AppFileUpload
              accept=".lrc,.txt"
              hint="LRC lyrics file"
              class="mt-3"
              @file-selected="onLrcFileSelected"
            />
            <AppButton
              variant="primary"
              block
              class="mt-3"
              :loading="uploading"
              :disabled="!pendingMediaFile"
              @click="uploadMedia"
            >Upload Media</AppButton>
          </div>
        </aside>
      </Transition>

      <!-- Main content -->
      <div class="playback-page__content">
        <button v-if="!sidebarOpen" class="playback-page__sidebar-toggle" @click="sidebarOpen = true" aria-label="Open media library">
          <svg width="20" height="20" viewBox="0 0 20 20" fill="none"><path d="M3 5h14M3 10h14M3 15h14" stroke="currentColor" stroke-width="1.5" stroke-linecap="round"/></svg>
        </button>

        <AppLoadingState v-if="mediaLoading" message="Loading media..." />
        <AppErrorState v-else-if="mediaError" :message="mediaError" retryable @retry="loadMedia" />

        <template v-else-if="currentTrack">
          <div class="playback-page__main">
            <!-- Left: Lyrics panel (60%) -->
            <div class="lyrics-panel">
              <div class="lyrics-panel__search">
                <AppInput
                  v-model="lyricsQuery"
                  placeholder="Search lyrics..."
                  @update:modelValue="debouncedSearch"
                >
                  <template #prefix>
                    <svg width="16" height="16" viewBox="0 0 16 16" fill="none"><circle cx="7" cy="7" r="5" stroke="currentColor" stroke-width="1.5"/><path d="M11 11l3 3" stroke="currentColor" stroke-width="1.5" stroke-linecap="round"/></svg>
                  </template>
                </AppInput>
              </div>

              <AppLoadingState v-if="lrcParsing" message="Parsing lyrics..." />

              <div v-else-if="parsedLyrics.length > 0" ref="lyricsContainer" class="lyrics-panel__lines">
                <div
                  v-for="(line, idx) in parsedLyrics"
                  :key="idx"
                  :ref="el => { if (el) lineRefs[idx] = el }"
                  class="lyrics-line"
                  :class="{
                    'lyrics-line--active': idx === activeLyricIndex,
                    'lyrics-line--pulse': idx === pulseLine,
                    'lyrics-line--search-match': searchMatchIndices.has(idx),
                  }"
                  @click="seekToLine(idx)"
                >
                  <span class="lyrics-line__time">{{ formatTime(line.time) }}</span>
                  <span v-if="line.words && line.words.length > 0" class="lyrics-line__text">
                    <span
                      v-for="(word, wi) in line.words"
                      :key="wi"
                      class="lyrics-word"
                      :class="{ 'lyrics-word--active': idx === activeLyricIndex && wi <= activeWordIndex }"
                    >{{ word.text }} </span>
                  </span>
                  <span v-else class="lyrics-line__text">{{ line.text }}</span>
                </div>
              </div>

              <div v-else class="lyrics-panel__empty">
                <svg width="56" height="56" viewBox="0 0 56 56" fill="none">
                  <circle cx="28" cy="28" r="24" stroke="currentColor" stroke-width="1.5"/>
                  <path d="M22 38V22l4-2v14M34 36V18l-4 2v14" stroke="currentColor" stroke-width="1.5" stroke-linecap="round"/>
                  <circle cx="20" cy="38" r="3" stroke="currentColor" stroke-width="1.5"/>
                  <circle cx="32" cy="36" r="3" stroke="currentColor" stroke-width="1.5"/>
                </svg>
                <p>Lyrics not available</p>
                <span class="text-sm text-muted">Upload an LRC file from the sidebar</span>
              </div>
            </div>

            <!-- Right: Player controls (40%) -->
            <div class="player-panel">
              <div class="player-panel__art">
                <img
                  v-if="currentTrack.coverUrl"
                  :src="currentTrack.coverUrl"
                  :alt="currentTrack.title"
                  crossorigin="anonymous"
                  ref="coverImg"
                  @load="extractDominantColor"
                />
                <div v-else class="player-panel__art-placeholder">
                  <svg width="64" height="64" viewBox="0 0 64 64" fill="none"><circle cx="32" cy="32" r="28" stroke="currentColor" stroke-width="1.5"/><path d="M26 42V22l16 10-16 10z" fill="currentColor" opacity="0.3"/></svg>
                </div>
              </div>

              <h3 class="player-panel__title">{{ currentTrack.title }}</h3>
              <p class="player-panel__artist">{{ currentTrack.artist || 'Unknown Artist' }}</p>

              <!-- Progress bar -->
              <div class="player-panel__progress">
                <span class="player-panel__time">{{ formatTime(currentTime) }}</span>
                <div class="player-panel__bar" @click="seekFromBar">
                  <div class="player-panel__bar-fill" :style="{ width: progressPercent + '%' }" />
                  <div class="player-panel__bar-thumb" :style="{ left: progressPercent + '%' }" />
                </div>
                <span class="player-panel__time">{{ formatTime(duration) }}</span>
              </div>

              <!-- Controls -->
              <div class="player-panel__controls">
                <button class="ctrl-btn" :class="{ 'ctrl-btn--active': shuffle }" @click="shuffle = !shuffle" aria-label="Shuffle">
                  <svg width="20" height="20" viewBox="0 0 20 20" fill="none"><path d="M3 14h2l3-4-3-4H3M13 6h2l-3 4 3 4h-2M7 6l6 8M7 14l6-8" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"/></svg>
                </button>
                <button class="ctrl-btn" @click="prevTrack" aria-label="Previous">
                  <svg width="22" height="22" viewBox="0 0 22 22" fill="none"><path d="M16 16L8 11l8-5v10z" fill="currentColor"/><path d="M6 6v10" stroke="currentColor" stroke-width="2" stroke-linecap="round"/></svg>
                </button>
                <button class="ctrl-btn ctrl-btn--play" @click="togglePlay" :aria-label="isPlaying ? 'Pause' : 'Play'">
                  <svg v-if="isPlaying" width="28" height="28" viewBox="0 0 28 28" fill="none"><rect x="7" y="6" width="4" height="16" rx="1" fill="currentColor"/><rect x="17" y="6" width="4" height="16" rx="1" fill="currentColor"/></svg>
                  <svg v-else width="28" height="28" viewBox="0 0 28 28" fill="none"><path d="M8 5v18l16-9L8 5z" fill="currentColor"/></svg>
                </button>
                <button class="ctrl-btn" @click="nextTrack" aria-label="Next">
                  <svg width="22" height="22" viewBox="0 0 22 22" fill="none"><path d="M6 6l8 5-8 5V6z" fill="currentColor"/><path d="M16 6v10" stroke="currentColor" stroke-width="2" stroke-linecap="round"/></svg>
                </button>
                <button class="ctrl-btn" :class="{ 'ctrl-btn--active': repeatMode !== 'none' }" @click="cycleRepeat" aria-label="Repeat">
                  <svg width="20" height="20" viewBox="0 0 20 20" fill="none"><path d="M4 9V7a4 4 0 014-4h4a4 4 0 014 4v6a4 4 0 01-4 4H8a4 4 0 01-4-4v-2" stroke="currentColor" stroke-width="1.5" stroke-linecap="round"/><path d="M7 6L4 9l3 3" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"/></svg>
                  <span v-if="repeatMode === 'one'" class="ctrl-btn__badge">1</span>
                </button>
              </div>

              <!-- Volume -->
              <div class="player-panel__volume">
                <button class="ctrl-btn" @click="toggleMute" aria-label="Mute">
                  <svg v-if="volume > 0" width="18" height="18" viewBox="0 0 18 18" fill="none"><path d="M3 7h2l4-4v12l-4-4H3V7z" fill="currentColor"/><path d="M13 6.5a4 4 0 010 5" stroke="currentColor" stroke-width="1.5" stroke-linecap="round"/></svg>
                  <svg v-else width="18" height="18" viewBox="0 0 18 18" fill="none"><path d="M3 7h2l4-4v12l-4-4H3V7z" fill="currentColor"/><path d="M13 7l4 4M17 7l-4 4" stroke="currentColor" stroke-width="1.5" stroke-linecap="round"/></svg>
                </button>
                <input type="range" min="0" max="1" step="0.01" :value="volume" class="volume-slider" @input="onVolumeChange" />
              </div>
            </div>
          </div>

          <!-- Bottom format chip -->
          <div class="playback-page__footer">
            <AppChip :label="formatLabel" variant="info" />
            <span v-if="unsupportedFormat" class="text-sm text-danger">Unsupported format -- playback may be limited</span>
          </div>
        </template>

        <AppEmptyState v-else title="No track selected" description="Choose a track from the library or upload new media." />
      </div>
    </div>

    <audio ref="audioEl" @timeupdate="onTimeUpdate" @ended="onTrackEnded" @loadedmetadata="onMetadataLoaded" @error="onAudioError" />
  </div>
</template>

<script setup>
import { ref, computed, watch, onBeforeUnmount, nextTick } from 'vue';
import * as playbackApi from '@/api/playback.js';
import AppButton from '@/components/common/AppButton.vue';
import AppInput from '@/components/common/AppInput.vue';
import AppChip from '@/components/common/AppChip.vue';
import AppFileUpload from '@/components/common/AppFileUpload.vue';
import AppLoadingState from '@/components/common/AppLoadingState.vue';
import AppErrorState from '@/components/common/AppErrorState.vue';
import AppEmptyState from '@/components/common/AppEmptyState.vue';
import AppToast from '@/components/common/AppToast.vue';

// ---- State ----
const toast = ref(null);
const audioEl = ref(null);
const coverImg = ref(null);
const lyricsContainer = ref(null);
const lineRefs = ref({});

const sidebarOpen = ref(true);
const mediaList = ref([]);
const mediaLoading = ref(false);
const mediaError = ref('');
const currentTrack = ref(null);
const uploading = ref(false);
const pendingMediaFile = ref(null);
const pendingLrcFile = ref(null);

const isPlaying = ref(false);
const currentTime = ref(0);
const duration = ref(0);
const volume = ref(0.8);
const prevVolume = ref(0.8);
const shuffle = ref(false);
const repeatMode = ref('none'); // none, all, one
const unsupportedFormat = ref(false);

const parsedLyrics = ref([]);
const lrcParsing = ref(false);
const activeLyricIndex = ref(-1);
const activeWordIndex = ref(-1);
const pulseLine = ref(-1);
const lyricsQuery = ref('');
const searchMatchIndices = ref(new Set());
const dominantColor = ref(null);

let debounceTimer = null;

// ---- Computed ----
const progressPercent = computed(() => (duration.value > 0 ? (currentTime.value / duration.value) * 100 : 0));

const formatLabel = computed(() => {
  if (!currentTrack.value) return '';
  const name = currentTrack.value.filename || currentTrack.value.title || '';
  const ext = name.split('.').pop()?.toLowerCase() || 'unknown';
  return ext.toUpperCase();
});

const pageBackground = computed(() => {
  if (dominantColor.value) {
    const c = dominantColor.value;
    return { background: `linear-gradient(135deg, rgba(${c.r},${c.g},${c.b},0.15) 0%, rgba(${c.r},${c.g},${c.b},0.03) 50%, transparent 100%)` };
  }
  return {};
});

// ---- LRC Parser ----
function parseLRC(text) {
  const lines = text.split('\n');
  const result = [];
  const lineRegex = /^\[(\d{2}):(\d{2})\.(\d{2,3})\](.*)/;
  const wordRegex = /<(\d{2}):(\d{2})\.(\d{2,3})>/g;

  for (const raw of lines) {
    const match = raw.match(lineRegex);
    if (!match) continue;
    const minutes = parseInt(match[1], 10);
    const seconds = parseInt(match[2], 10);
    const ms = parseInt(match[3].padEnd(3, '0'), 10);
    const time = minutes * 60 + seconds + ms / 1000;
    const content = match[4].trim();

    // Check for word-level timestamps
    const words = [];
    let lastIdx = 0;
    let wordMatch;
    const clean = content.replace(wordRegex, (m, wm, ws, wms, offset) => {
      const wordTime = parseInt(wm, 10) * 60 + parseInt(ws, 10) + parseInt(wms.padEnd(3, '0'), 10) / 1000;
      if (lastIdx < offset) {
        const text = content.substring(lastIdx, offset).replace(/<[^>]*>/g, '').trim();
        if (text) words.push({ time: words.length === 0 ? time : wordTime, text });
      }
      lastIdx = offset + m.length;
      return '';
    });

    if (lastIdx < content.length) {
      const remainder = content.substring(lastIdx).replace(/<[^>]*>/g, '').trim();
      if (remainder) {
        if (words.length === 0) {
          words.push({ time, text: remainder });
        } else {
          words.push({ time: words[words.length - 1].time, text: remainder });
        }
      }
    }

    const lineText = content.replace(/<[^>]*>/g, '').trim();
    result.push({
      time,
      text: lineText,
      words: words.length > 1 ? words : [],
    });
  }

  result.sort((a, b) => a.time - b.time);
  return result;
}

// ---- Data loading ----
async function loadMedia() {
  mediaLoading.value = true;
  mediaError.value = '';
  try {
    const { data } = await playbackApi.getMedia();
    mediaList.value = Array.isArray(data) ? data : data.items || [];
  } catch (err) {
    mediaError.value = err.message || 'Failed to load media';
  } finally {
    mediaLoading.value = false;
  }
}

function selectTrack(track) {
  currentTrack.value = track;
  unsupportedFormat.value = false;
  parsedLyrics.value = [];
  activeLyricIndex.value = -1;
  activeWordIndex.value = -1;
  currentTime.value = 0;
  duration.value = 0;
  isPlaying.value = false;
  dominantColor.value = null;

  if (audioEl.value) {
    audioEl.value.src = track.url || track.streamUrl || '';
    audioEl.value.volume = volume.value;
    audioEl.value.load();
  }

  if (track.lrcContent) {
    lrcParsing.value = true;
    nextTick(() => {
      parsedLyrics.value = parseLRC(track.lrcContent);
      lrcParsing.value = false;
    });
  } else if (track.id) {
    loadLyrics(track.id);
  }
}

async function loadLyrics(mediaId) {
  lrcParsing.value = true;
  try {
    const { data } = await playbackApi.parseLyrics(mediaId);
    const content = typeof data === 'string' ? data : data.lrc || data.content || '';
    parsedLyrics.value = parseLRC(content);
  } catch {
    parsedLyrics.value = [];
  } finally {
    lrcParsing.value = false;
  }
}

// ---- Playback controls ----
function togglePlay() {
  if (!audioEl.value) return;
  if (isPlaying.value) {
    audioEl.value.pause();
    isPlaying.value = false;
  } else {
    audioEl.value.play().then(() => { isPlaying.value = true; }).catch(() => {});
  }
}

function prevTrack() {
  if (mediaList.value.length === 0) return;
  const idx = mediaList.value.findIndex(t => t.id === currentTrack.value?.id);
  const prev = idx > 0 ? idx - 1 : mediaList.value.length - 1;
  selectTrack(mediaList.value[prev]);
  nextTick(() => audioEl.value?.play().then(() => { isPlaying.value = true; }).catch(() => {}));
}

function nextTrack() {
  if (mediaList.value.length === 0) return;
  const idx = mediaList.value.findIndex(t => t.id === currentTrack.value?.id);
  let next;
  if (shuffle.value) {
    next = Math.floor(Math.random() * mediaList.value.length);
  } else {
    next = idx < mediaList.value.length - 1 ? idx + 1 : 0;
  }
  selectTrack(mediaList.value[next]);
  nextTick(() => audioEl.value?.play().then(() => { isPlaying.value = true; }).catch(() => {}));
}

function cycleRepeat() {
  const modes = ['none', 'all', 'one'];
  const idx = modes.indexOf(repeatMode.value);
  repeatMode.value = modes[(idx + 1) % modes.length];
}

function toggleMute() {
  if (volume.value > 0) {
    prevVolume.value = volume.value;
    volume.value = 0;
  } else {
    volume.value = prevVolume.value || 0.8;
  }
  if (audioEl.value) audioEl.value.volume = volume.value;
}

function onVolumeChange(e) {
  volume.value = parseFloat(e.target.value);
  if (audioEl.value) audioEl.value.volume = volume.value;
}

function seekFromBar(e) {
  if (!audioEl.value || duration.value === 0) return;
  const rect = e.currentTarget.getBoundingClientRect();
  const pct = (e.clientX - rect.left) / rect.width;
  audioEl.value.currentTime = pct * duration.value;
}

function seekToLine(idx) {
  const line = parsedLyrics.value[idx];
  if (!line || !audioEl.value) return;
  audioEl.value.currentTime = line.time;
  if (!isPlaying.value) {
    audioEl.value.play().then(() => { isPlaying.value = true; }).catch(() => {});
  }
  pulseLine.value = idx;
  setTimeout(() => { pulseLine.value = -1; }, 200);
}

// ---- Audio events ----
function onTimeUpdate() {
  if (!audioEl.value) return;
  currentTime.value = audioEl.value.currentTime;
  updateActiveLyric();
}

function onMetadataLoaded() {
  if (audioEl.value) duration.value = audioEl.value.duration;
}

function onTrackEnded() {
  isPlaying.value = false;
  if (repeatMode.value === 'one') {
    audioEl.value.currentTime = 0;
    audioEl.value.play().then(() => { isPlaying.value = true; }).catch(() => {});
  } else if (repeatMode.value === 'all') {
    nextTrack();
  }
}

function onAudioError() {
  unsupportedFormat.value = true;
  isPlaying.value = false;
  toast.value?.addToast({ message: 'Audio playback error. Format may be unsupported.', type: 'warning' });
}

// ---- Lyrics sync ----
function updateActiveLyric() {
  const t = currentTime.value;
  let idx = -1;
  for (let i = parsedLyrics.value.length - 1; i >= 0; i--) {
    if (parsedLyrics.value[i].time <= t) { idx = i; break; }
  }
  if (idx !== activeLyricIndex.value) {
    activeLyricIndex.value = idx;
    scrollToActiveLine(idx);
  }
  // Word-level
  if (idx >= 0 && parsedLyrics.value[idx].words?.length > 0) {
    let wi = -1;
    for (let w = parsedLyrics.value[idx].words.length - 1; w >= 0; w--) {
      if (parsedLyrics.value[idx].words[w].time <= t) { wi = w; break; }
    }
    activeWordIndex.value = wi;
  } else {
    activeWordIndex.value = -1;
  }
}

function scrollToActiveLine(idx) {
  if (idx < 0 || !lineRefs.value[idx]) return;
  lineRefs.value[idx].scrollIntoView({ behavior: 'smooth', block: 'center' });
}

// ---- Lyrics search ----
function debouncedSearch() {
  clearTimeout(debounceTimer);
  debounceTimer = setTimeout(() => searchLyrics(), 300);
}

function searchLyrics() {
  const q = lyricsQuery.value.trim().toLowerCase();
  if (!q) { searchMatchIndices.value = new Set(); return; }
  const matches = new Set();
  parsedLyrics.value.forEach((line, idx) => {
    if (line.text.toLowerCase().includes(q)) matches.add(idx);
  });
  searchMatchIndices.value = matches;
}

// ---- Dynamic theme color ----
function extractDominantColor() {
  const img = coverImg.value;
  if (!img) return;
  try {
    const canvas = document.createElement('canvas');
    canvas.width = 1;
    canvas.height = 1;
    const ctx = canvas.getContext('2d');
    ctx.drawImage(img, 0, 0, 1, 1);
    const [r, g, b] = ctx.getImageData(0, 0, 1, 1).data;
    dominantColor.value = { r, g, b };
  } catch {
    dominantColor.value = { r: 26, g: 115, b: 232 };
  }
}

// ---- Upload ----
function onMediaFileSelected(file) { pendingMediaFile.value = file; }
function onLrcFileSelected(file) { pendingLrcFile.value = file; }

async function uploadMedia() {
  if (!pendingMediaFile.value) return;
  uploading.value = true;
  try {
    const form = new FormData();
    form.append('file', pendingMediaFile.value);
    if (pendingLrcFile.value) form.append('lrc', pendingLrcFile.value);
    await playbackApi.createMedia(form);
    toast.value?.addToast({ message: 'Media uploaded successfully', type: 'success' });
    pendingMediaFile.value = null;
    pendingLrcFile.value = null;
    await loadMedia();
  } catch (err) {
    toast.value?.addToast({ message: err.message || 'Upload failed', type: 'error' });
  } finally {
    uploading.value = false;
  }
}

// ---- Helpers ----
function formatTime(sec) {
  if (!sec || isNaN(sec)) return '0:00';
  const m = Math.floor(sec / 60);
  const s = Math.floor(sec % 60);
  return `${m}:${s.toString().padStart(2, '0')}`;
}

// ---- Lifecycle ----
loadMedia();

onBeforeUnmount(() => {
  clearTimeout(debounceTimer);
  if (audioEl.value) { audioEl.value.pause(); audioEl.value.src = ''; }
});
</script>

<style lang="scss" scoped>
.playback-page {
  min-height: 100vh;
  transition: background $transition-slow;

  &__top {
    display: flex;
    min-height: calc(100vh - 48px);
  }

  &__sidebar {
    width: 300px;
    min-width: 300px;
    background: $color-neutral-0;
    border-right: 1px solid $border-color;
    display: flex;
    flex-direction: column;
    height: calc(100vh - 48px);
    position: sticky;
    top: 0;
  }

  &__sidebar-toggle {
    position: absolute;
    top: $space-4;
    left: $space-4;
    z-index: 10;
    width: 36px;
    height: 36px;
    display: flex;
    align-items: center;
    justify-content: center;
    border-radius: $border-radius-base;
    background: $color-neutral-0;
    border: 1px solid $border-color;
    color: $color-neutral-600;
    box-shadow: $shadow-sm;
    transition: all $transition-fast;
    &:hover { background: $color-neutral-50; }
  }

  &__content {
    flex: 1;
    min-width: 0;
    padding: $space-6;
    position: relative;
  }

  &__main {
    display: flex;
    gap: $space-6;
    @media (max-width: 900px) { flex-direction: column; }
  }

  &__footer {
    display: flex;
    align-items: center;
    gap: $space-3;
    margin-top: $space-4;
    padding-top: $space-4;
    border-top: 1px solid $border-color;
  }
}

// Sidebar internals
.sidebar {
  &__header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: $space-4 $space-5;
    border-bottom: 1px solid $border-color;
    h4 { font-size: $font-size-md; color: $color-neutral-800; }
  }
  &__close {
    width: 28px; height: 28px; display: flex; align-items: center; justify-content: center;
    border-radius: $border-radius-base; color: $color-neutral-400;
    &:hover { background: $color-neutral-50; color: $color-neutral-600; }
  }
  &__list {
    flex: 1; overflow-y: auto; padding: $space-2;
  }
  &__track {
    display: flex; align-items: center; gap: $space-3; padding: $space-2 $space-3;
    border-radius: $border-radius-base; cursor: pointer; transition: background $transition-fast;
    &:hover { background: $color-neutral-50; }
    &--active { background: $color-primary-50; }
  }
  &__track-art {
    width: 40px; height: 40px; border-radius: $border-radius-base; overflow: hidden; flex-shrink: 0;
    img { width: 100%; height: 100%; object-fit: cover; }
  }
  &__track-art-placeholder {
    width: 40px; height: 40px; display: flex; align-items: center; justify-content: center;
    background: $color-neutral-100; border-radius: $border-radius-base; color: $color-neutral-400;
  }
  &__track-info { min-width: 0; display: flex; flex-direction: column; }
  &__track-title {
    font-size: $font-size-base; font-weight: $font-weight-medium; color: $color-neutral-800;
    white-space: nowrap; overflow: hidden; text-overflow: ellipsis;
  }
  &__track-artist {
    font-size: $font-size-xs; color: $color-neutral-500;
  }
  &__upload { padding: $space-4; border-top: 1px solid $border-color; }
}

// Lyrics panel
.lyrics-panel {
  flex: 0 0 60%;
  max-width: 60%;
  display: flex;
  flex-direction: column;
  @media (max-width: 900px) { flex: 1; max-width: 100%; }

  &__search { margin-bottom: $space-4; }

  &__lines {
    flex: 1;
    max-height: 60vh;
    overflow-y: auto;
    padding: $space-4;
    background: rgba($color-neutral-0, 0.7);
    border-radius: $border-radius-lg;
    border: 1px solid $border-color;
    scroll-behavior: smooth;
  }

  &__empty {
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    padding: $space-12;
    color: $color-neutral-400;
    p { font-size: $font-size-lg; margin-top: $space-4; font-weight: $font-weight-medium; color: $color-neutral-500; }
  }
}

.lyrics-line {
  display: flex;
  align-items: baseline;
  gap: $space-3;
  padding: $space-2 $space-3;
  border-radius: $border-radius-base;
  cursor: pointer;
  transition: all $transition-fast;

  &:hover { background: $color-neutral-50; }

  &__time {
    font-size: $font-size-xs;
    color: $color-neutral-400;
    font-family: $font-family-mono;
    flex-shrink: 0;
    width: 40px;
  }
  &__text {
    font-size: $font-size-base;
    color: $color-neutral-600;
    line-height: $line-height-relaxed;
  }

  &--active {
    background: $color-primary-50;
    .lyrics-line__text { color: $color-primary-700; font-size: $font-size-lg; font-weight: $font-weight-semibold; }
    .lyrics-line__time { color: $color-primary-500; }
  }

  &--pulse {
    animation: lyric-pulse 200ms ease;
  }

  &--search-match {
    background: $color-warning-50;
    .lyrics-line__text { color: $color-warning-700; }
  }
}

.lyrics-word {
  transition: color $transition-fast;
  &--active { color: $color-primary-600; font-weight: $font-weight-bold; }
}

@keyframes lyric-pulse {
  0% { transform: scale(1); }
  50% { transform: scale(1.02); background: $color-primary-100; }
  100% { transform: scale(1); }
}

// Player panel
.player-panel {
  flex: 0 0 40%;
  max-width: 40%;
  display: flex;
  flex-direction: column;
  align-items: center;
  padding: $space-6;
  background: rgba($color-neutral-0, 0.8);
  border-radius: $border-radius-lg;
  border: 1px solid $border-color;
  @media (max-width: 900px) { flex: 1; max-width: 100%; }

  &__art {
    width: 220px; height: 220px; border-radius: $border-radius-lg;
    overflow: hidden; box-shadow: $shadow-lg; margin-bottom: $space-5;
    img { width: 100%; height: 100%; object-fit: cover; }
  }
  &__art-placeholder {
    width: 100%; height: 100%; display: flex; align-items: center; justify-content: center;
    background: $color-neutral-100; color: $color-neutral-300;
  }

  &__title { font-size: $font-size-xl; font-weight: $font-weight-bold; color: $color-neutral-900; text-align: center; margin: 0; }
  &__artist { font-size: $font-size-base; color: $color-neutral-500; margin: $space-1 0 $space-5; }

  &__progress {
    width: 100%;
    display: flex;
    align-items: center;
    gap: $space-3;
    margin-bottom: $space-5;
  }
  &__time { font-size: $font-size-xs; color: $color-neutral-500; font-family: $font-family-mono; width: 36px; text-align: center; }
  &__bar {
    flex: 1; height: 6px; background: $color-neutral-200; border-radius: $border-radius-full;
    cursor: pointer; position: relative;
  }
  &__bar-fill { height: 100%; background: $color-primary-500; border-radius: $border-radius-full; transition: width 100ms linear; }
  &__bar-thumb {
    position: absolute; top: 50%; transform: translate(-50%, -50%);
    width: 14px; height: 14px; background: $color-primary-500; border: 2px solid $color-neutral-0;
    border-radius: $border-radius-full; box-shadow: $shadow-sm; transition: left 100ms linear;
  }

  &__controls { display: flex; align-items: center; gap: $space-4; margin-bottom: $space-5; }

  &__volume {
    display: flex; align-items: center; gap: $space-2; width: 100%;
  }
}

.ctrl-btn {
  width: 40px; height: 40px; display: flex; align-items: center; justify-content: center;
  border-radius: $border-radius-full; color: $color-neutral-600; position: relative;
  transition: all $transition-fast;
  &:hover { background: $color-neutral-100; color: $color-neutral-800; }
  &--play {
    width: 52px; height: 52px; background: $color-primary-500; color: #fff;
    &:hover { background: $color-primary-600; color: #fff; }
  }
  &--active { color: $color-primary-500; }
  &__badge {
    position: absolute; top: 2px; right: 2px; font-size: 9px; font-weight: $font-weight-bold;
    color: $color-primary-600; line-height: 1;
  }
}

.volume-slider {
  flex: 1; height: 4px; -webkit-appearance: none; appearance: none;
  background: $color-neutral-200; border-radius: $border-radius-full; outline: none;
  &::-webkit-slider-thumb {
    -webkit-appearance: none; width: 14px; height: 14px; border-radius: 50%;
    background: $color-primary-500; cursor: pointer; border: 2px solid $color-neutral-0;
    box-shadow: $shadow-xs;
  }
}
</style>
