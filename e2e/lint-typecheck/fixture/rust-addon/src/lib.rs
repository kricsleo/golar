use golar::*;

fn check_call<'a>(ctx: &RuleContext<'a>, node: CallExpression<'a>) {
    let Some(typ) = node
        .expression()
        .and_then(|expression| ctx.program.get_type_at_location(&expression))
    else {
        return;
    };

    if let Some(t) = typ.cast::<IntrinsicType>()
        && t.intrinsic_name() == "any"
    {
        ctx.report_node(node.as_node(), "Unsafe any call.");
    }
}

fn run<'a>(ctx: &RuleContext<'a>) {
    walk(ctx.source_file.as_node(), &mut |node: Node<'_>| {
        if let Some(call) = node.cast::<CallExpression>() {
            check_call(ctx, call);
        }

        false
    });
}

inventory::submit! {
    Rule {
        name: "rust/unsafe-calls",
        run,
    }
}
