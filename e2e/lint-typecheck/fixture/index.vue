<script lang="ts" setup>
	import { fn } from './index.ts'

	declare function onlyStr(str: string): string
</script>

<template>
	{{ onlyStr(123) }}
	{{ fn() }}
</template>
