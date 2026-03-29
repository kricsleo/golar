<script lang="ts" setup>
	import { useRoute } from 'vue-router'

	const route = useRoute()
	route.params.unknown
	const id = route.params.id

	type Equal<X, Y> =
		(<T>() => T extends X ? 1 : 2) extends <T>() => T extends Y ? 1 : 2
			? true
			: false

	const res: Equal<typeof id, string> = true
</script>
