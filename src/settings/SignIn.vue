<script setup lang="ts">
// The owner's ways in, as GlimStone's sign-in tile draws them. Until the owner is signed in, the tab is the way in
// itself. While CastleParcel is open with an upload password, that password comes first, since it guards the
// sign-in's changes too.
import { SignInTile } from '@glimstone/vue';
import { takeSignInCode, type SignInTileCard } from '@glimstone/core';
import { IDENTITY_PROVIDERS } from '@glimstone/core/identityProviders';
import { computed } from 'vue';
import { config } from '@/src/config';
import { locale } from '@/src/i18n';
import Gate from '@/src/screens/Gate.vue';
import { signIn } from '@/src/signin/kit';
import SignInPage from '@/src/signin/SignInPage.vue';
import { account } from '@/src/signin/state';
import { tileText } from '@/src/signin/words';
import { toast } from '@/src/toast';
import { gate } from '@/src/upload';

const CARDS: SignInTileCard[] = ['password', 'askPassword', 'twoFactor', 'passkeys', 'sessions', 'tokens', 'services'];
const KEYS = {
  change: 'IconPencil', setPassword: 'IconKey', removePassword: 'IconKeyRevoke', savePassword: 'IconSave', setUp: 'IconShieldOn',
  newCodes: 'IconRefresh', turnOff: 'IconShieldOff', copy: 'IconCopy', confirm: 'IconConfirm', add: 'IconAdd', rename: 'IconPencil',
  remove: 'IconTrash', end: 'IconSignOut', endOthers: 'IconSignOut', createToken: 'IconAdd', revoke: 'IconKeyRevoke',
  addService: 'IconAdd', test: 'IconSendTest', edit: 'IconPencil', turnOn: 'IconSignIn', save: 'IconSave', passkey: 'IconKey',
  token: 'IconKey', laptop: 'IconLaptop', phone: 'IconPhone', service: 'IconSignIn', forward: 'IconForward',
};

const text = computed(tileText);
// The sign-in card takes the code a sign-in service sent back while nobody is signed in.
const arrived = account.open || account.signedIn ? takeSignInCode() : null;
const time = (iso: string) => new Date(iso).toLocaleString(locale(), { dateStyle: 'medium', timeStyle: 'short' });
const guarded = computed(() => account.open && config.uploadProtected && !gate.token);
</script>

<template>
  <div class="cp-signin-tab">
    <SignInPage v-if="!account.open && !account.signedIn" />
    <Gate v-else-if="guarded" />
    <SignInTile
      v-else
      :api="signIn"
      :text="text"
      :lang="locale()"
      :providers="IDENTITY_PROVIDERS"
      :cards="CARDS"
      :scopes="['read', 'all']"
      :notify="toast"
      :arrived="arrived"
      :time="time"
      :label-keys="KEYS"
    />
  </div>
</template>
