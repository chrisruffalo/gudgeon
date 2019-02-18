Name:       {{{ git_name }}}
Version:    {{{ git_version }}}
Release:    1%{?dist}
Summary:    A blocking DNS proxy/cache with host/subnet level rules and other features for managing home or small office DNS.

License:    MIT
URL:        https://github.com/chrisruffalo/gudgeon
VCS:        {{{ git_vcs }}}
Source:     {{{ git_pack }}}

BuildRequires: git, make, automake, gcc, gcc-c++, upx, curl

%description
A blocking DNS proxy/cache with host/subnet level rules and other features for managing home or small office DNS.

%global debug_package %{nil}

%prep
# rpkg macro for setup
{{{ git_dir_setup_macro }}}
# install go if not provided
if ! which go; then
    export LOCALARCH=$(uname -m | sed 's/x86_64/amd64/' | sed 's/i686/386/' | sed 's/686/386/' | sed 's/i386/386/')
    export REMARCHIVE=go1.11.5.linux-$LOCALARCH.tar.gz
    curl https://dl.google.com/go/$REMARCHIVE -L -o /tmp/$REMARCHIVE
    mkdir /tmp/golang
    tar xvf /tmp/$REMARCHIVE -C /tmp/golang
    mv /tmp/golang/go /usr/local/go
    rm -rf /tmp/golang /tmp/$REMARCHIVE
    ln -s /usr/local/go/bin/go /usr/bin/go
fi

%build
export VERSION="{{{ git_version }}}"
export GITHASH=""
export GOOS_LIST="linux" 
export GARCH_LIST="$(uname -m)" 
make prepare
make download
make build

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
