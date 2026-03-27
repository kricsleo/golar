macro_rules! golar_host_symbols {
    ($callback:ident) => {
        $callback! {
            fn golar_workspace_get_requested_file(
                workspace: *const Workspace,
                file_idx: u32,
            ) -> FileWithProgram;

            fn golar_program_get_type_at_location(
                program: *const RawProgram,
                node: *const RawNode,
            ) -> *const RawType;

            fn golar_workspace_report(
                workspace: *const Workspace,
                file_idx: u32,
                start: i32,
                end: i32,
                rule_name: *const c_char,
                rule_name_len: usize,
                message: *const c_char,
                message_len: usize,
            );
        }
    };
}

#[cfg(not(windows))]
#[path = "host_symbols_linked.rs"]
mod imp;

#[cfg(windows)]
#[path = "host_symbols_windows.rs"]
mod imp;

pub(crate) use imp::*;
