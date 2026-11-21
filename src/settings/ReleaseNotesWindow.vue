<script setup lang="ts">
// The notes of one release in a window of the app. The page may only talk to
// its own server, which fetches them from GitHub.
import { BusyRing, Button, Window, type WindowAction } from '@glimstone/vue';
import { computed, onMounted, ref } from 'vue';
import { notesBlocks, type Block } from '@/lib/notes';
import { t } from '@/src/i18n';
import { windowWords } from '@/src/ui';

const props = defineProps<{ version: string; repo: string }>();
const emit = defineEmits<{ close: [] }>();

const page = computed(() => `${props.repo}/releases/tag/v${props.version}`);
const notes = ref<Block[] | null>(null);
const failed = ref(false);

async function load() {
  failed.value = false;
  notes.value = null;
  try {
    const res = await fetch(`/api/release-notes/v${encodeURIComponent(props.version)}`);
    if (!res.ok) throw new Error(String(res.status));
    notes.value = notesBlocks(((await res.json()) as { notes?: string }).notes ?? '');
  } catch {
    failed.value = true;
  }
}
onMounted(load);

const actions = computed<WindowAction[]>(() => [
  { label: t('info.notesOnGitHub'), labelKey: 'IconGithub', onClick: () => { window.open(page.value, '_blank', 'noopener'); } },
]);
</script>

<template>
  <Window :title="t('info.notesTitle', { version })" :actions="actions" :text="windowWords()" size="wide" @close="emit('close')">
    <p v-if="!notes && !failed" class="cp-notes-wait"><BusyRing size="small" /> {{ t('info.notesLoading') }}</p>
    <div v-else-if="failed" class="cp-notes-failed">
      <p>{{ t('info.notesFailed') }}</p>
      <Button :label="t('info.notesRetry')" label-key="IconRetry" @click="load" />
    </div>
    <div v-else class="cp-notes">
      <template v-for="(block, i) in notes" :key="i">
        <h3 v-if="block.kind === 'heading'" class="cp-notes-heading">
          <template v-for="(part, j) in block.parts" :key="j"><a v-if="part.href" :href="part.href" target="_blank" rel="noopener noreferrer">{{ part.text }}</a><strong v-else-if="part.bold">{{ part.text }}</strong><code v-else-if="part.code">{{ part.text }}</code><template v-else>{{ part.text }}</template></template>
        </h3>
        <ul v-else-if="block.kind === 'list'" class="cp-notes-list">
          <li v-for="(item, j) in block.items" :key="j">
            <template v-for="(part, k) in item" :key="k"><a v-if="part.href" :href="part.href" target="_blank" rel="noopener noreferrer">{{ part.text }}</a><strong v-else-if="part.bold">{{ part.text }}</strong><code v-else-if="part.code">{{ part.text }}</code><template v-else>{{ part.text }}</template></template>
          </li>
        </ul>
        <p v-else class="cp-notes-text">
          <template v-for="(part, j) in block.parts" :key="j"><a v-if="part.href" :href="part.href" target="_blank" rel="noopener noreferrer">{{ part.text }}</a><strong v-else-if="part.bold">{{ part.text }}</strong><code v-else-if="part.code">{{ part.text }}</code><template v-else>{{ part.text }}</template></template>
        </p>
      </template>
    </div>
  </Window>
</template>

<style scoped>
.cp-notes { display: flex; flex-direction: column; gap: .6rem; line-height: 1.55; }
.cp-notes-heading { margin: .6rem 0 0; font-size: var(--text-title, 1rem); font-weight: 600; color: var(--carbon-text); }
.cp-notes-heading:first-child { margin-top: 0; }
.cp-notes-text, .cp-notes-list { margin: 0; color: var(--carbon-text-sub); }
.cp-notes-list { padding-inline-start: 1.25rem; display: flex; flex-direction: column; gap: .3rem; }
.cp-notes strong { color: var(--carbon-text); font-weight: 600; }
.cp-notes a { color: var(--accent-ink); text-decoration: underline; }
.cp-notes code { font-family: var(--font-mono, ui-monospace, monospace); font-size: .9em; overflow-wrap: anywhere; }
.cp-notes-wait { display: flex; align-items: center; gap: .5rem; margin: 0; color: var(--carbon-text-sub); }
.cp-notes-failed { display: flex; flex-direction: column; align-items: flex-start; gap: .75rem; }
.cp-notes-failed p { margin: 0; color: var(--carbon-text-sub); }
</style>
