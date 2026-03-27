use crate::ast_generated::*;
use crate::common::*;
use crate::host_symbols;

use std::collections::HashSet;

use napi::bindgen_prelude::BigInt;
use napi_derive::napi;

#[repr(C)]
pub struct Workspace {
    _data: (),
    _marker: core::marker::PhantomData<(*mut u8, core::marker::PhantomPinned)>,
}

#[repr(C)]
pub struct RawProgram {
    _data: (),
    _marker: core::marker::PhantomData<(*mut u8, core::marker::PhantomPinned)>,
}

#[repr(C)]
pub struct FileWithProgram {
    program: *const RawProgram,
    source_file: *const RawNode,
}

pub struct Program<'a> {
    _raw: *const RawProgram,
    _marker: core::marker::PhantomData<&'a ()>,
}

impl<'a> Program<'a> {
    pub fn get_type_at_location(&self, node: &Node<'a>) -> Option<Type<'a>> {
        let t = host_symbols::golar_program_get_type_at_location(self._raw, node.raw);
        if t == core::ptr::null() {
            return None;
        }
        return Some(Type::from_raw(t));
    }
}

pub fn walk<'a, F>(node: Node<'a>, visitor: &mut F) -> bool
where
    F: FnMut(Node<'a>) -> bool,
{
    if visitor(node) {
        return true;
    }
    node.for_each_child(&mut |child| walk(child, visitor))
}

pub struct RuleContext<'a> {
    workspace: *const Workspace,
    file_idx: u32,
    rule_name: &'static str,
    pub program: Program<'a>,
    pub source_file: SourceFile<'a>,
}

impl<'a> RuleContext<'a> {
    pub fn report(&self, start: i32, end: i32, message: &str) {
        host_symbols::golar_workspace_report(
            self.workspace,
            self.file_idx,
            start,
            end,
            self.rule_name.as_ptr().cast(),
            self.rule_name.len(),
            message.as_ptr().cast(),
            message.len(),
        );
    }

    pub fn report_node(&self, node: Node<'a>, message: &str) {
        self.report(node.pos(), node.end(), message);
    }
}

pub struct Rule {
    pub name: &'static str,
    pub run: for<'a> fn(ctx: &'a RuleContext<'a>) -> (),
}

pub use inventory;
inventory::collect!(Rule);

#[napi]
#[allow(dead_code)]
fn setup(golar_addon_path: String) -> napi::Result<()> {
    host_symbols::setup(&golar_addon_path)
}

#[napi]
#[allow(dead_code)]
fn lint(workspace_ptr: BigInt, file_idx: u32, rule_names: Vec<String>) -> napi::Result<String> {
    host_symbols::verify_ready()?;

    let ws = workspace_ptr.get_u64().1 as *const Workspace;
    let file_with_program = host_symbols::golar_workspace_get_requested_file(ws, file_idx);
    let source_file =
        unsafe { SourceFile::from_node_unchecked(Node::from_raw(file_with_program.source_file)) };
    let selected_rule_names = rule_names
        .iter()
        .map(|name| name.as_str())
        .collect::<HashSet<_>>();
    let mut matched_rule_names = HashSet::<&str>::new();

    for rule in inventory::iter::<Rule> {
        if !selected_rule_names.contains(rule.name) {
            continue;
        }
        matched_rule_names.insert(rule.name);

        let ctx = RuleContext {
            workspace: ws,
            file_idx,
            rule_name: rule.name,
            program: Program {
                _raw: file_with_program.program,
                _marker: core::marker::PhantomData,
            },
            source_file,
        };
        (rule.run)(&ctx);
    }

    if matched_rule_names.len() != selected_rule_names.len() {
        let mut missing_rule_names = rule_names
            .into_iter()
            .filter(|name| !matched_rule_names.contains(name.as_str()))
            .collect::<Vec<_>>();
        missing_rule_names.sort();
        return Err(napi::Error::from_reason(format!(
            "missing native rules: {}",
            missing_rule_names.join(", "),
        )));
    }

    return Ok(String::from("ok"));
}
