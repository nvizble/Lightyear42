#!/bin/bash
# lightyear born2beroot checks. Run INSIDE the evaluated VM, as root:
#   sudo bash born2beroot.sh            (readable report)
#   LT_RAW=1 bash born2beroot.sh        (lightyear's protocol, one line per case:
#                                        R<TAB>group<TAB>name<TAB>OK|KO|SKIP<TAB>detail)
# The login defaults to the hostname without the trailing 42; pass it as $1
# to check it explicitly.

LANG=C
LC_ALL=C
export LANG LC_ALL
PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin:$PATH

GROUP=""
PASS=0
FAIL=0
SKIPPED=0

group() { GROUP="$1"; [ -z "$LT_RAW" ] && printf '\n\033[1m%s\033[0m\n' "$1"; }

emit() { # status name detail
	local status="$1" name="$2" detail="$3"
	detail=$(printf '%s' "$detail" | tr '\t' ' ' | awk 'BEGIN{ORS=" ⏎ "} {print}' | sed 's/ ⏎ $//')
	case "$status" in
		OK) PASS=$((PASS + 1)) ;;
		SKIP) SKIPPED=$((SKIPPED + 1)) ;;
		*) FAIL=$((FAIL + 1)) ;;
	esac
	if [ -n "$LT_RAW" ]; then
		printf 'R\t%s\t%s\t%s\t%s\n' "$GROUP" "$name" "$status" "$detail"
		return
	fi
	case "$status" in
		OK) printf '  \033[32m✓\033[0m %s\n' "$name" ;;
		SKIP) printf '  \033[90m– %s (%s)\033[0m\n' "$name" "$detail" ;;
		*) printf '  \033[31m✗ %s\033[0m\n      %s\n' "$name" "$(printf '%s' "$detail" | sed 's/ ⏎ /\n      /g')" ;;
	esac
}

ok() { emit OK "$1" ""; }
ko() { emit KO "$1" "$2"; }
skip() { emit SKIP "$1" "$2"; }
check() { # name detail-if-failed command...
	local name="$1" detail="$2"
	shift 2
	if "$@" >/dev/null 2>&1; then ok "$name"; else ko "$name" "$detail"; fi
}
has() { command -v "$1" >/dev/null 2>&1; }

ROOT=0
[ "$(id -u)" = 0 ] && ROOT=1
needroot() { # name
	[ "$ROOT" = 1 ] && return 0
	skip "$1" "rode como root: sudo lightyear test born2beroot"
	return 1
}

OS_ID=""
OS_VERSION=""
if [ -r /etc/os-release ]; then
	OS_ID=$(. /etc/os-release; echo "$ID")
	OS_VERSION=$(. /etc/os-release; echo "$PRETTY_NAME")
fi
ROCKY=0
[ "$OS_ID" = rocky ] && ROCKY=1

HOST=$(hostname 2>/dev/null)
LOGIN="${1:-${HOST%42}}"

# ---------------------------------------------------------------- system --
group "sistema"
if [ "$(uname -s)" != Linux ]; then
	ko "Linux" "não é Linux: rode dentro da VM do avaliado"
	exit 1
fi
case "$OS_ID" in
	debian)
		if grep -qiE 'sid|testing|unstable' /etc/debian_version /etc/apt/sources.list /etc/apt/sources.list.d/* 2>/dev/null; then
			ko "Debian estável (sem testing/unstable)" "$OS_VERSION; as fontes do apt citam testing/unstable/sid"
		else
			ok "Debian estável: $OS_VERSION"
		fi ;;
	rocky) ok "Rocky: $OS_VERSION" ;;
	*) ko "Debian ou Rocky" "sistema: ${OS_VERSION:-desconhecido}" ;;
esac

gui=""
for bin in Xorg X Xwayland weston gnome-shell plasmashell sway; do
	has "$bin" && gui="$gui $bin"
done
if has dpkg-query; then
	pkgs=$(dpkg-query -W -f='${db:Status-Abbrev} ${Package}\n' 'xserver-xorg*' xwayland weston gdm3 lightdm sddm 2>/dev/null | awk '$1 == "ii" {print $2}' | tr '\n' ' ')
	gui="$gui $pkgs"
elif has rpm; then
	gui="$gui $(rpm -qa 'xorg-x11-server*' xorg-x11-server-Xwayland weston gdm sddm 2>/dev/null | tr '\n' ' ')"
fi
gui=$(echo $gui)
if [ -z "$gui" ]; then ok "sem interface gráfica (X.org/Wayland)"; else ko "sem interface gráfica (X.org/Wayland)" "instalado: $gui (vale nota 0)"; fi

# ------------------------------------------------------------ partitions --
group "partições"
if has lsblk; then
	types=$(lsblk -rno TYPE 2>/dev/null)
	crypts=$(printf '%s\n' "$types" | grep -cx crypt)
	lvms=$(printf '%s\n' "$types" | grep -cx lvm)
	tree=$(lsblk -o NAME,TYPE,SIZE,MOUNTPOINT 2>/dev/null)
	if [ "$crypts" -ge 1 ]; then ok "partição criptografada (LUKS)"; else ko "partição criptografada (LUKS)" "nenhum volume crypt no lsblk:
$tree"; fi
	if [ "$lvms" -ge 2 ]; then ok "pelo menos 2 partições LVM ($lvms)"; else ko "pelo menos 2 partições LVM" "$lvms volume(s) LVM:
$tree"; fi
	inside=$(lsblk -rno TYPE,NAME -s 2>/dev/null | awk '$1 == "lvm" {lv=1} $1 == "crypt" && lv {n++} END {print n+0}')
	if [ "$crypts" -ge 1 ] && [ "$lvms" -ge 2 ]; then
		ok "os volumes LVM ficam dentro da partição criptografada"
	else
		ko "os volumes LVM ficam dentro da partição criptografada" "esperado: disco → part → crypt → lvm (veja lsblk)"
	fi
	: "$inside"
else
	skip "partições" "lsblk não encontrado"
fi

# ------------------------------------------------------ AppArmor / SELinux --
if [ "$ROCKY" = 1 ]; then
	group "SELinux"
	if has getenforce; then
		mode=$(getenforce 2>/dev/null)
		if [ "$mode" = Enforcing ]; then ok "SELinux ativo (Enforcing)"; else ko "SELinux ativo (Enforcing)" "getenforce: $mode"; fi
	else
		ko "SELinux ativo (Enforcing)" "getenforce não encontrado"
	fi
	if grep -qE '^SELINUX=enforcing' /etc/selinux/config 2>/dev/null; then ok "SELinux ligado na inicialização"; else ko "SELinux ligado na inicialização" "/etc/selinux/config não tem SELINUX=enforcing"; fi
else
	group "AppArmor"
	if [ "$(cat /sys/module/apparmor/parameters/enabled 2>/dev/null)" = Y ]; then
		ok "AppArmor ativo no kernel"
	else
		ko "AppArmor ativo no kernel" "/sys/module/apparmor/parameters/enabled não é Y"
	fi
	if has systemctl; then
		check "AppArmor ligado na inicialização" "systemctl is-enabled apparmor: $(systemctl is-enabled apparmor 2>&1)" systemctl is-enabled apparmor
	fi
	if needroot "perfis do AppArmor carregados"; then
		if has aa-status && aa-status --enabled 2>/dev/null; then ok "perfis do AppArmor carregados"; else ko "perfis do AppArmor carregados" "aa-status: $(aa-status 2>&1 | head -2)"; fi
	fi
fi

# -------------------------------------------------------------------- SSH --
group "SSH"
sshd_unit=ssh
[ "$ROCKY" = 1 ] && sshd_unit=sshd
if has systemctl; then
	check "serviço SSH rodando" "systemctl is-active $sshd_unit: $(systemctl is-active $sshd_unit 2>&1)" systemctl is-active "$sshd_unit"
	check "SSH liga na inicialização" "systemctl is-enabled $sshd_unit: $(systemctl is-enabled $sshd_unit 2>&1)" systemctl is-enabled "$sshd_unit"
fi
listening=$(ss -Htln 2>/dev/null | awk '{print $4}')
if printf '%s\n' "$listening" | grep -qE ':4242$'; then ok "escutando na porta 4242"; else ko "escutando na porta 4242" "portas TCP abertas: $(printf '%s ' $listening)"; fi
if printf '%s\n' "$listening" | grep -qE ':22$'; then ko "nada na porta 22" "ainda tem algo escutando na 22"; else ok "nada na porta 22"; fi
if needroot "PermitRootLogin no"; then
	conf=$(sshd -T 2>/dev/null)
	if [ -z "$conf" ]; then
		conf=$(cat /etc/ssh/sshd_config /etc/ssh/sshd_config.d/*.conf 2>/dev/null | grep -viE '^\s*#' | tr 'A-Z' 'a-z')
	fi
	root_login=$(printf '%s\n' "$conf" | awk 'tolower($1) == "permitrootlogin" {print tolower($2); exit}')
	if [ "$root_login" = no ]; then ok "PermitRootLogin no"; else ko "PermitRootLogin no" "PermitRootLogin é ${root_login:-o padrão (prohibit-password)}: tem que ser no"; fi
	port=$(printf '%s\n' "$conf" | awk 'tolower($1) == "port" {print $2}' | tr '\n' ' ')
	if [ "$(echo $port)" = 4242 ]; then ok "Port 4242 na configuração"; else ko "Port 4242 na configuração" "Port: ${port:-22 (padrão)}"; fi
fi

# --------------------------------------------------------------- firewall --
if [ "$ROCKY" = 1 ]; then
	group "firewalld"
	if has firewall-cmd; then
		if [ "$(firewall-cmd --state 2>&1)" = running ]; then ok "firewalld ativo"; else ko "firewalld ativo" "firewall-cmd --state: $(firewall-cmd --state 2>&1)"; fi
		check "firewalld liga na inicialização" "systemctl is-enabled firewalld" systemctl is-enabled firewalld
		if needroot "só a porta 4242 aberta"; then
			ports=$(firewall-cmd --list-ports 2>/dev/null)
			services=$(firewall-cmd --list-services 2>/dev/null)
			extra=$(printf '%s %s' "$ports" "$services" | tr ' ' '\n' | grep -v -e '^4242/tcp$' -e '^$' -e '^cockpit$' -e '^dhcpv6-client$' | tr '\n' ' ')
			if printf '%s' "$ports" | grep -q '4242/tcp' && [ -z "$extra" ]; then ok "só a porta 4242 aberta"; else ko "só a porta 4242 aberta" "portas: ${ports:-nenhuma} · serviços: ${services:-nenhum} (no bônus, outras portas podem ser permitidas)"; fi
		fi
	else
		ko "firewalld instalado" "firewall-cmd não encontrado"
	fi
else
	group "UFW"
	if has ufw; then
		has systemctl && check "UFW liga na inicialização" "systemctl is-enabled ufw: $(systemctl is-enabled ufw 2>&1)" systemctl is-enabled ufw
		if needroot "UFW ativo"; then
			status=$(ufw status 2>&1)
			if printf '%s\n' "$status" | grep -q 'Status: active'; then ok "UFW ativo"; else ko "UFW ativo" "$status"; fi
			rules=$(printf '%s\n' "$status" | awk '/^--/ {r=1; next} r && NF {print $1}' | sed 's/(v6)//' | sort -u)
			extra=$(printf '%s\n' "$rules" | grep -vE '^4242(/tcp)?$' | tr '\n' ' ')
			if printf '%s\n' "$rules" | grep -qE '^4242(/tcp)?$' && [ -z "$(echo $extra)" ]; then
				ok "só a porta 4242 aberta"
			else
				ko "só a porta 4242 aberta" "regras: $(printf '%s ' $rules) (no bônus, outras portas podem ser permitidas)"
			fi
		fi
	else
		ko "UFW instalado" "ufw não encontrado"
	fi
fi

# ------------------------------------------------------- hostname / users --
group "hostname e usuários"
if printf '%s' "$HOST" | grep -qE '42$'; then ok "hostname termina em 42 ($HOST)"; else ko "hostname termina em 42" "hostname: $HOST"; fi
if [ -n "$1" ]; then
	if [ "$HOST" = "${1}42" ]; then ok "hostname é ${1}42"; else ko "hostname é ${1}42" "hostname: $HOST"; fi
fi
if id "$LOGIN" >/dev/null 2>&1; then
	ok "usuário $LOGIN existe"
	groups_of=$(id -nG "$LOGIN")
	sudo_group=sudo
	[ "$ROCKY" = 1 ] && sudo_group=wheel
	for g in user42 "$sudo_group"; do
		if printf ' %s ' "$groups_of" | grep -q " $g "; then ok "$LOGIN está no grupo $g"; else ko "$LOGIN está no grupo $g" "grupos de $LOGIN: $groups_of"; fi
	done
else
	ko "usuário $LOGIN existe" "não há usuário $LOGIN (o login vem do hostname; passe outro: bash born2beroot.sh <login>)"
fi
if getent group user42 >/dev/null; then ok "grupo user42 existe"; else ko "grupo user42 existe" "getent group user42 não achou"; fi

# ---------------------------------------------------------- password policy --
group "política de senha"
defs() { awk -v k="$1" '$1 == k {v=$2} END {print v}' /etc/login.defs 2>/dev/null; }
[ "$(defs PASS_MAX_DAYS)" = 30 ] && ok "PASS_MAX_DAYS 30 (login.defs)" || ko "PASS_MAX_DAYS 30 (login.defs)" "PASS_MAX_DAYS = $(defs PASS_MAX_DAYS)"
[ "$(defs PASS_MIN_DAYS)" = 2 ] && ok "PASS_MIN_DAYS 2 (login.defs)" || ko "PASS_MIN_DAYS 2 (login.defs)" "PASS_MIN_DAYS = $(defs PASS_MIN_DAYS)"
[ "$(defs PASS_WARN_AGE)" = 7 ] && ok "PASS_WARN_AGE 7 (login.defs)" || ko "PASS_WARN_AGE 7 (login.defs)" "PASS_WARN_AGE = $(defs PASS_WARN_AGE)"
for u in root "$LOGIN"; do
	if ! id "$u" >/dev/null 2>&1; then continue; fi
	if needroot "validade da senha de $u (chage)"; then
		line=$(awk -F: -v u="$u" '$1 == u {print $4" "$5" "$6}' /etc/shadow)
		set -- $line
		if [ "$1" = 2 ] && [ "$2" = 30 ] && [ "$3" = 7 ]; then
			ok "validade da senha de $u: mín 2, máx 30, aviso 7"
		else
			ko "validade da senha de $u: mín 2, máx 30, aviso 7" "chage de $u: mín ${1:-?}, máx ${2:-?}, aviso ${3:-?} (o login.defs só vale para usuários novos: chage -m 2 -M 30 -W 7 $u)"
		fi
	fi
done

pam=/etc/pam.d/common-password
[ "$ROCKY" = 1 ] && pam=/etc/pam.d/system-auth
pamline=$(grep -E '^[^#]*pam_pwquality\.so' "$pam" 2>/dev/null | head -1)
if [ -n "$pamline" ]; then ok "pam_pwquality ativo em $pam"; else ko "pam_pwquality ativo em $pam" "nenhuma linha com pam_pwquality.so (apt install libpam-pwquality)"; fi
# Options may live on the PAM line or in pwquality.conf (and .conf.d).
opts=$(printf '%s\n' "$pamline" | tr ' \t' '\n\n' | grep -v pam_ ; cat /etc/security/pwquality.conf /etc/security/pwquality.conf.d/*.conf 2>/dev/null | grep -vE '^\s*#' | tr -d ' \t')
opt() { printf '%s\n' "$opts" | awk -F= -v k="$1" '$1 == k {v=($2 == "" ? "yes" : $2)} END {print v}'; }
num_check() { # name key op value hint
	local v
	v=$(opt "$2")
	if [ -n "$v" ] && [ "$v" -eq "$v" ] 2>/dev/null && [ "$v" "$3" "$4" ]; then ok "$1"; else ko "$1" "$2 = ${v:-não definido} $5"; fi
}
num_check "mínimo de 10 caracteres" minlen -ge 10 "(minlen=10)"
num_check "pelo menos uma maiúscula" ucredit -le -1 "(ucredit=-1)"
num_check "pelo menos uma minúscula" lcredit -le -1 "(lcredit=-1)"
num_check "pelo menos um número" dcredit -le -1 "(dcredit=-1)"
v=$(opt maxrepeat)
if [ -n "$v" ] && [ "$v" -ge 1 ] 2>/dev/null && [ "$v" -le 3 ]; then ok "no máximo 3 caracteres iguais seguidos"; else ko "no máximo 3 caracteres iguais seguidos" "maxrepeat = ${v:-não definido} (maxrepeat=3)"; fi
[ -n "$(opt reject_username)" ] && ok "não pode conter o nome do usuário" || ko "não pode conter o nome do usuário" "falta reject_username"
num_check "7 caracteres diferentes da senha anterior" difok -ge 7 "(difok=7)"
[ -n "$(opt enforce_for_root)" ] && ok "a política vale para o root" || ko "a política vale para o root" "falta enforce_for_root"

# ------------------------------------------------------------------- sudo --
group "sudo"
if has sudo; then ok "sudo instalado"; else ko "sudo instalado" "sudo não encontrado"; fi
if needroot "configuração do sudo"; then
	rules=$(cat /etc/sudoers /etc/sudoers.d/* 2>/dev/null | grep -vE '^\s*#' )
	defaults=$(printf '%s\n' "$rules" | grep -E '^\s*Defaults' | sed 's/^\s*Defaults[^ \t]*\s*//' | tr ',' '\n' | sed 's/^\s*//;s/\s*$//')
	d() { printf '%s\n' "$defaults" | grep -E "^$1" | tail -1; }
	v=$(d 'passwd_tries' | sed 's/.*=\s*//')
	[ "$v" = 3 ] && ok "3 tentativas de senha (passwd_tries=3)" || ko "3 tentativas de senha (passwd_tries=3)" "passwd_tries = ${v:-não definido (o padrão é 3, mas o subject pede configurar)}"
	v=$(d 'badpass_message')
	[ -n "$v" ] && ok "mensagem de senha errada própria" || ko "mensagem de senha errada própria" "falta Defaults badpass_message=\"...\""
	logfile=$(d 'logfile' | sed 's/.*=\s*//;s/"//g')
	iolog=$(d 'iolog_dir' | sed 's/.*=\s*//;s/"//g')
	case "$logfile" in
		/var/log/sudo/*) ok "log do sudo em /var/log/sudo/ ($logfile)" ;;
		*) ko "log do sudo em /var/log/sudo/" "logfile = ${logfile:-não definido}" ;;
	esac
	if d 'log_input' | grep -q . && d 'log_output' | grep -q .; then ok "registra entradas e saídas (log_input, log_output)"; else ko "registra entradas e saídas (log_input, log_output)" "falta Defaults log_input e/ou log_output"; fi
	case "$iolog" in
		/var/log/sudo*) ok "iolog_dir em /var/log/sudo ($iolog)" ;;
		*) ko "iolog_dir em /var/log/sudo" "iolog_dir = ${iolog:-não definido (o padrão é /var/log/sudo-io)}" ;;
	esac
	d 'requiretty' | grep -q . && ok "requiretty" || ko "requiretty" "falta Defaults requiretty"
	sp=$(d 'secure_path' | sed 's/.*=\s*//;s/"//g')
	if [ -n "$sp" ] && ! printf '%s' "$sp" | grep -qE '(^|:)\.?(:|$)'; then ok "secure_path restrito ($sp)"; else ko "secure_path restrito" "secure_path = ${sp:-não definido}"; fi
	[ -d /var/log/sudo ] && ok "a pasta /var/log/sudo existe" || ko "a pasta /var/log/sudo existe" "não existe"
fi

# ---------------------------------------------------------- monitoring.sh --
group "monitoring.sh"
cronlines=""
if needroot "monitoring.sh no cron"; then
	cronlines=$( { crontab -l -u root 2>/dev/null; cat /etc/crontab /etc/cron.d/* 2>/dev/null; } | grep -vE '^\s*#' | grep monitoring)
	if [ -n "$cronlines" ]; then ok "monitoring.sh no cron"; else ko "monitoring.sh no cron" "nenhuma linha com monitoring no crontab do root nem em /etc/cron.d"; fi
	if printf '%s\n' "$cronlines" | grep -qE '^\s*\*/10\s|^\s*(0,10,20,30,40,50)\s'; then ok "a cada 10 minutos"; else ko "a cada 10 minutos" "linhas: $cronlines"; fi
	if printf '%s\n' "$cronlines" | grep -q '@reboot' || { has systemctl && systemctl list-unit-files 2>/dev/null | grep -qi monitoring; }; then
		ok "roda ao ligar o servidor"
	else
		ko "roda ao ligar o servidor" "sem @reboot no cron (o subject: \"At server startup, the script will display…\")"
	fi
fi
if has systemctl; then
	cron_unit=cron
	[ "$ROCKY" = 1 ] && cron_unit=crond
	check "cron ativo" "systemctl is-active $cron_unit" systemctl is-active "$cron_unit"
fi
script=$(printf '%s\n' "$cronlines" | grep -oE '/[^ ;&|]*monitoring\.sh' | head -1)
[ -z "$script" ] && for p in /usr/local/bin/monitoring.sh /root/monitoring.sh /home/"$LOGIN"/monitoring.sh; do [ -f "$p" ] && script=$p && break; done
if [ -z "$script" ] || [ ! -f "$script" ]; then
	ko "o script existe" "monitoring.sh não encontrado (procurei no cron, /usr/local/bin, /root, /home/$LOGIN)"
else
	ok "o script existe ($script)"
	first=$(head -1 "$script")
	case "$first" in
		'#!/bin/bash'*|'#!/usr/bin/bash'*|'#!/usr/bin/env bash'*) ok "escrito em bash ($first)" ;;
		*) ko "escrito em bash" "a 1ª linha é $first" ;;
	esac
	grep -q 'wall' "$script" && ok "usa o wall" || ko "usa o wall" "o subject pede mostrar em todos os terminais (wall)"
	if needroot "roda sem erros"; then
		fake=$(mktemp -d)
		printf '#!/bin/sh\nif [ $# -gt 0 ] && [ "$1" != -n ]; then printf "%%s\\n" "$*"; else [ "$1" = -n ] && shift; [ $# -gt 0 ] && printf "%%s\\n" "$*" || cat; fi\n' > "$fake/wall"
		chmod +x "$fake/wall"
		out=$(PATH="$fake:$PATH" timeout 30 bash "$script" 2>"$fake/err")
		code=$?
		err=$(cat "$fake/err")
		rm -rf "$fake"
		if [ "$code" = 0 ] && [ -z "$err" ]; then ok "roda sem erros"; else ko "roda sem erros" "saída $code; stderr: $err"; fi
		line() { printf '%s\n' "$out" | grep -iE "$1" | head -1; }
		# Each item: label pattern, then what must be in that line.
		l=$(line 'arch'); if printf '%s' "$l" | grep -qF "$(uname -r)" && printf '%s' "$l" | grep -qF "$(uname -m)"; then ok "arquitetura e kernel"; else ko "arquitetura e kernel" "esperado $(uname -r) e $(uname -m), recebido: ${l:-nada}"; fi
		pcpu=$(grep 'physical id' /proc/cpuinfo | sort -u | wc -l)
		want="$pcpu"
		[ "$pcpu" = 0 ] && want="[01]" # no "physical id" in /proc/cpuinfo: 0 or 1 are both fair
		l=$(line 'physical'); if printf '%s' "$l" | grep -qE "(^|[^0-9])$want([^0-9]|$)"; then ok "CPUs físicas ($pcpu)"; else ko "CPUs físicas" "esperado $pcpu, recebido: ${l:-nada}"; fi
		vcpu=$(grep -c '^processor' /proc/cpuinfo)
		l=$(line 'vcpu|virtual'); if printf '%s' "$l" | grep -qE "(^|[^0-9])$vcpu([^0-9]|$)"; then ok "vCPUs ($vcpu)"; else ko "vCPUs" "esperado $vcpu, recebido: ${l:-nada}"; fi
		l=$(line 'mem|ram'); if printf '%s' "$l" | grep -qE '[0-9]+(\.[0-9]+)?%'; then ok "uso de memória (com %)"; else ko "uso de memória (com %)" "recebido: ${l:-nada}"; fi
		total=$(free -m | awk '/^Mem:/ {print $2}')
		if printf '%s' "$l" | grep -qE "(^|[^0-9])$total([^0-9]|$)|(^|[^0-9])$((total - 1))([^0-9]|$)|(^|[^0-9])$((total + 1))([^0-9]|$)"; then ok "memória total ($total MB)"; else ko "memória total" "esperado $total MB (free -m), recebido: ${l:-nada}"; fi
		l=$(line 'disk|storage'); if printf '%s' "$l" | grep -qE '[0-9]+(\.[0-9]+)?%'; then ok "uso de disco (com %)"; else ko "uso de disco (com %)" "recebido: ${l:-nada}"; fi
		l=$(line 'cpu load|cpu usage|load'); if printf '%s' "$l" | grep -qE '[0-9]+(\.[0-9]+)?%'; then ok "uso de CPU (com %)"; else ko "uso de CPU (com %)" "recebido: ${l:-nada}"; fi
		boot=$(who -b 2>/dev/null | awk '{print $3" "$4}')
		l=$(line 'boot'); if [ -n "$boot" ] && printf '%s' "$l" | grep -qF "$boot"; then ok "último boot ($boot)"; else ko "último boot" "esperado $boot (who -b), recebido: ${l:-nada}"; fi
		want=no
		lsblk -rno TYPE 2>/dev/null | grep -qx lvm && want=yes
		l=$(line 'lvm'); if printf '%s' "$l" | grep -qiw "$want"; then ok "LVM em uso ($want)"; else ko "LVM em uso" "esperado $want, recebido: ${l:-nada}"; fi
		l=$(line 'tcp|connexion|connection'); if printf '%s' "$l" | grep -qE '[0-9]'; then ok "conexões TCP"; else ko "conexões TCP" "recebido: ${l:-nada}"; fi
		l=$(line 'user'); if printf '%s' "$l" | grep -qE '[0-9]'; then ok "usuários logados"; else ko "usuários logados" "recebido: ${l:-nada}"; fi
		ip4=$(hostname -I 2>/dev/null | awk '{print $1}')
		mac=$(ip link 2>/dev/null | awk '/link\/ether/ {print $2; exit}')
		l=$(line 'network|ip'); if printf '%s' "$l" | grep -qF "$ip4" && printf '%s' "$l" | grep -qiF "$mac"; then ok "IPv4 e MAC ($ip4, $mac)"; else ko "IPv4 e MAC" "esperado $ip4 e $mac, recebido: ${l:-nada}"; fi
		l=$(line 'sudo'); if printf '%s' "$l" | grep -qE '[0-9]'; then ok "comandos sudo"; else ko "comandos sudo" "recebido: ${l:-nada}"; fi
	fi
fi

if [ -z "$LT_RAW" ]; then
	printf '\n\033[1mborn2beroot:\033[0m \033[32m%d ok\033[0m · \033[31m%d falharam\033[0m · %d pulados\n' "$PASS" "$FAIL" "$SKIPPED"
	[ "$ROOT" = 0 ] && printf 'Vários testes precisam de root: rode com sudo.\n'
fi
[ "$FAIL" = 0 ]
