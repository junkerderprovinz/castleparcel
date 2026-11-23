<script setup lang="ts">
// The bar over every page: the way back at the start of the line, the way into Settings at its end. The start page
// itself needs no way back, but the options of a parcel and its finished link lead back to it from the same spot.
import { Button } from '@glimstone/vue';
import { computed, ref, watchEffect } from 'vue';
import { config } from '@/src/config';
import { t } from '@/src/i18n';
import { follow, go, route } from '@/src/router';
import { signOut } from '@/src/signin/kit';
import { account } from '@/src/signin/state';
import { leaveStep } from '@/src/steps';
import { send } from '@/src/upload';

const props = defineProps<{ screen: string }>();
const inSettings = computed(() => props.screen === 'settings');

// Settings leads back to where the person came from, every other page to the start page.
const before = ref('/');
watchEffect(() => {
  if (route.parts[0] !== 'settings') before.value = location.pathname + location.hash;
});
const back = computed(() => (inSettings.value ? before.value : '/'));

// The owner's door sits beside Settings on the pages that send, never on a link someone was given. Without an upload
// password those pages show the sign-in card themselves.
const sending = computed(() => ['home', 'note', 'request', 'settings'].includes(props.screen));
const signInShown = computed(() => sending.value && !inSettings.value && !account.open && !account.signedIn && config.uploadProtected);
const signOutShown = computed(() => sending.value && account.signedIn);
</script>

<template>
  <header class="cp-head">
    <Button v-if="screen !== 'home'" :label="t('app.back')" label-key="IconBack" :href="back" data-cp="back" @click="follow($event, back)" />
    <Button v-else-if="send.stage === 'options'" :label="t('app.back')" label-key="IconBack" data-cp="back" @click="leaveStep" />
    <Button v-else-if="send.stage === 'done' && send.sent" :label="t('app.leave')" label-key="IconSignOut" data-cp="leave" @click="leaveStep" />
    <span class="cp-spacer"></span>
    <Button v-if="signInShown" variant="icon" :label="t('signin.signIn')" label-key="IconSignIn" data-cp="signin-open" @click="go('/settings/signin')" />
    <Button v-if="signOutShown" variant="icon" :label="t('signin.logout')" label-key="IconSignOut" data-cp="signout" @click="signOut(t('signin.bye'))" />
    <Button v-if="!inSettings" variant="icon" :label="t('app.settings')" label-key="IconGear" data-cp="settings" @click="go('/settings/general')" />
  </header>
</template>
