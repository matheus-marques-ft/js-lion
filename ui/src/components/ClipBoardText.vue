<script lang="ts" setup>
import { computed, ref } from 'vue';
import { readClipboardText } from '@/utils/clipboard';
import { useDebounceFn } from '@vueuse/core';
import { NInput } from 'naive-ui';
import CardContainer from '@/components/CardContainer/index.vue';
const emit = defineEmits(['update:text']);
import { useMessage } from 'naive-ui';
import { useI18n } from 'vue-i18n';
const { t } = useI18n();
// Internal input value
const inputValue = ref<string>('');
const isLoading = ref<boolean>(false);
const message = useMessage();
const props = defineProps<{
  remoteText?: string;
  disabled?: boolean;
  copyDisabled?: boolean;
  pasteDisabled?: boolean;
  pastePolicyDisabled?: boolean;
  textLimit?: number;
}>();

const showRemoteText = ref<boolean>(false);
const maxlength = computed(() => {
  return props.textLimit && props.textLimit > 0 ? props.textLimit : undefined;
});

const getTextLength = (text: string) => Array.from(text).length;

const validateTextLimit = (text: string) => {
  if (props.pastePolicyDisabled) {
    message.warning(t('ClipboardPasteDeniedByPolicy'));
    return false;
  }
  if (props.disabled || props.pasteDisabled) {
    message.warning(`${t('Paste')} ${t('NoPermission')}`);
    return false;
  }
  if (maxlength.value && getTextLength(text) > maxlength.value) {
    message.warning(`${t('Paste')} ${t('ClipboardTextLimitExceeded')}: ${maxlength.value}`);
    return false;
  }
  return true;
};

// Manually read clipboard content
const loadClipboardText = async () => {
  try {
    isLoading.value = true;
    const text = await readClipboardText();
    if (!validateTextLimit(text)) {
      return;
    }
    inputValue.value = text;
    await handleInput(text);
  } catch (error) {
    console.log('Failed to read clipboard text:', error);
    // A user-friendly error message could be added here
  } finally {
    isLoading.value = false;
  }
};

// Handle input event
const handleInput = useDebounceFn((value: string) => {
  if (!validateTextLimit(value)) {
    return;
  }
  emit('update:text', value);
}, 300);

// Handle focus event, try to auto-read the clipboard
const handleFocus = async () => {
  // Only auto-read when the input is empty
  if (!inputValue.value.trim()) {
    try {
      await loadClipboardText();
    } catch (error) {
      // Silently handle the error, don't affect user experience
      console.debug('Auto-read clipboard failed, user can click button to read manually');
    }
  }
};

const noSideSpace = (value: string) => {
  return !value.startsWith(' ') && !value.endsWith(' ') && !value.startsWith('\n');
};

const size = {
  minRows: 4,
  maxRows: 6,
};
</script>

<template>
  <CardContainer :title="t('Clipboard')">
    <n-form-item :label="t('ShowRemoteClip')" label-placement="left">
      <n-switch v-model:value="showRemoteText" :disabled="props.disabled || props.copyDisabled" />
    </n-form-item>
    <n-input
      v-model:value="inputValue"
      @input="handleInput"
      @focus="handleFocus"
      type="textarea"
      :allow-input="noSideSpace"
      :autosize="size"
      :maxlength="maxlength"
      show-count
      clearable
      :placeholder="t('AutoPasteOnClick')"
      :disabled="props.disabled || props.pasteDisabled"
    >
    </n-input>
    <n-form-item v-if="showRemoteText">
      <n-input
        :value="props.remoteText"
        type="textarea"
        :autosize="size"
        readonly
        placeholder=""
        show-count
        :disabled="props.disabled || props.copyDisabled"
      />
    </n-form-item>
  </CardContainer>
</template>

<!-- <template>
  <n-card class="w-full" :title="t('Clipboard')">
    <n-input
      v-model:value="inputValue"
      @input="handleInput"
      @focus="handleFocus"
      type="textarea"
      :allow-input="noSideSpace"
      :autosize="size"
      :maxlength="maxlength"
      show-count
      clearable
      :placeholder="t('AutoPasteOnClick')"
      :disabled="props.disabled"
    >
    </n-input>
  </n-card> -->
<!-- <n-space vertical> -->

<!-- <n-space> -->
<!-- <n-button
        @click="loadClipboardText"
        type="primary"
        size="small"
      >
       Paste from clipboard
      </n-button> -->
<!-- <n-button
        @click="loadRemoteClipboardText"
        type="primary"
        size="small"
        :disabled="props.disabled"
      >
        Show remote-synced clipboard content</n-button
      > -->
<!-- </n-space> -->
<!-- <n-input
      v-if="showRemoteText"
      :value="props.remoteText"
      type="textarea"
      :autosize="size"
      readonly
      placeholder="Remote-synced clipboard content"
      :disabled="props.disabled"
    /> -->
<!-- </n-space> -->
<!-- </template> -->
