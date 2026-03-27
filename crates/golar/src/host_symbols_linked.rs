use crate::bindings::{FileWithProgram, RawProgram, Workspace};
use crate::common::{RawNode, RawType};

use std::ffi::c_char;

mod linked {
    use super::*;

    macro_rules! declare_linked_imports {
        ($(fn $name:ident($($arg:ident: $arg_ty:ty),* $(,)?) $(-> $ret:ty)?;)+) => {
            unsafe extern "C" {
                $(
                    pub(super) safe fn $name($($arg: $arg_ty),*) $(-> $ret)?;
                )+
            }
        };
    }

    golar_host_symbols!(declare_linked_imports);
}

macro_rules! define_linked_wrappers {
    ($(fn $name:ident($($arg:ident: $arg_ty:ty),* $(,)?) $(-> $ret:ty)?;)+) => {
        $(
            pub(crate) fn $name($($arg: $arg_ty),*) $(-> $ret)? {
                linked::$name($($arg),*)
            }
        )+
    };
}

pub(crate) fn setup(_golar_addon_path: &str) -> napi::Result<()> {
    Ok(())
}

pub(crate) fn verify_ready() -> napi::Result<()> {
    Ok(())
}

golar_host_symbols!(define_linked_wrappers);
