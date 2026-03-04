<template>
	<div v-if="false"></div>
	<template v-else>
		<!-- @vue-expect-error -->
		<Comp :foo="() => 'foo'" />
	</template>
</template>

<script setup lang="ts">
import Comp from './comp.vue'
</script>
