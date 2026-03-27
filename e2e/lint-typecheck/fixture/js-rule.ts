import { defineRule } from 'golar/unstable'
import {
	isCallExpression,
	TypeFlags,
	type CallExpression,
} from 'golar/unstable-tsgo'

export const jsRule = defineRule({
	name: 'js/unsafe-calls',
	setup(context) {
		function checkCall(node: CallExpression) {
			const type = context.program.getTypeAtLocation(node.expression)
			if (type == null) {
				return
			}
			if ((type.flags & TypeFlags.Any) !== 0 && type.intrinsicName === 'any') {
				context.report({
					message: 'Unsafe any call.',
					range: {
						begin: node.expression.pos,
						end: node.expression.end,
					},
				})
			}
		}

		context.sourceFile.forEachChild(function visit(node) {
			if (isCallExpression(node)) {
				checkCall(node)
			}
			node.forEachChild(visit)
		})
	},
})
