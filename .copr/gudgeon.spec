Name:       {{{ git_name }}}
Version:    {{{ git_version }}}
Release:    1%{?dist}
Summary:    A blocking DNS proxy/cache with host/subnet level rules and other features for managing home or small office DNS.

License:    MIT
URL:        https://github.com/chrisruffalo/gudgeon
VCS:        {{{ git_vcs }}}
Source:     {{{ git_pack }}}

BuildRequires: golang >= 1.11, make, automake, gcc, gcc-c++, upx

%description
A blocking DNS proxy/cache with host/subnet level rules and other features for managing home or small office DNS.

%global debug_package %{nil}

%prep
{{{ git_dir_setup_macro }}}

%build
make prepare
make download
VERSION="{{{ git_version }}}" GITHASH="" GOOS_LIST="linux" GARCH_LIST="$(uname -m)" make build

%install
%make_install

%files
%license LICENSE
/bin/gudgeon
%config(noreplace) /etc/gudgeon/gudgeon.yml
%config(noreplace) /etc/gudgeon
/var/lib/gudgeon
%config(noreplace) /lib/systemd/system/gudgeon.socket
%config(noreplace) /lib/systemd/system/gudgeon.service

%pre
USER_EXISTS=$(id -u gudgeon)
if [[ "0" != "$?" ]]; then
    useradd gudgeon -s /sbin/nologin || true
fi

%post
# change ownership of directories
chown -R :gudgeon /etc/gudgeon
chown -R gudgeon:gudgeon /var/lib/gudgeon

# mod gudgeon user for files created/owned by install
usermod gudgeon -d /var/lib/gudgeon || true

# reload daemon files
systemctl daemon-reload

# restart service only if it is already running to pick up the new version
IS_RUNNING=$(systemctl is-active gudgeon)
if [[ "active" == "${IS_RUNNING}" ]]; then
    systemctl restart gudgeon
fi
