# SPDX-License-Identifier: BSD-3-Clause
# Private experimental overlay: no public source fetch or upstream claim.
PORTNAME=	horvium
CATEGORIES=	sysutils
MASTER_SITES=
FETCH_CMD=	${FALSE}

MAINTAINER=	cameronjstone@icloud.com
COMMENT=	Experimental read-only Horvium lab management package
WWW=		https://github.com/gonz0w/horvium

LICENSE=	BSD3CLAUSE BSD0CLAUSE
LICENSE_COMB=	multi
LICENSE_FILE_BSD3CLAUSE=	${WRKSRC}/LICENSE
LICENSE_FILE_BSD0CLAUSE=	${WRKSRC}/third_party/htmx-LICENSE.txt

# This generated local binding is deliberately separate from the source archive.
# Native qualification remains mandatory; this is not a production release tag.
.include "source.mk"
DISTNAME=	horvium-${DISTVERSION}-${HORVIUM_SOURCE_REV}
DIST_SUBDIR=	horvium

ONLY_FOR_ARCHS=	amd64
USES=		go:1.27,modules
GO_TARGET=	./cmd/horvium:${PREFIX}/sbin/horvium
CGO_ENABLED=	0
GO_GOPROXY=	off
GO_GOSUMDB=	off
GO_ENV+=	GOTOOLCHAIN=local GOENV=off GOWORK=off GOTELEMETRY=off
# GO_MODULE is intentionally unset: the authenticated private bundle is vendored.
# Defining it would select the framework's module-proxy fetch/tidy path.

USE_RC_SUBR=	horvium_helper horvium
SUB_FILES=	horvium_rc.subr pkg-message pkg-pre-install pkg-pre-deinstall

.include <bsd.port.pre.mk>

.if ${PREFIX} != /usr/local
IGNORE=	this first lab profile requires PREFIX=/usr/local
.endif

post-install:
	${MKDIR} ${STAGEDIR}${PREFIX}/libexec/horvium ${STAGEDIR}${DOCSDIR}
	${INSTALL_SCRIPT} ${WRKDIR}/horvium_rc.subr ${STAGEDIR}${PREFIX}/libexec/horvium/rc.subr
	${INSTALL_DATA} ${FILESDIR}/horvium.json.sample ${STAGEDIR}${PREFIX}/etc/horvium.json.sample
	${INSTALL_DATA} ${WRKSRC}/LICENSE ${WRKSRC}/NOTICE.md ${WRKSRC}/PORT-SOURCE.json ${STAGEDIR}${DOCSDIR}
	${INSTALL_DATA} ${WRKSRC}/third_party/xsys-LICENSE.txt ${WRKSRC}/third_party/htmx-LICENSE.txt ${STAGEDIR}${DOCSDIR}
	${INSTALL_DATA} ${FILESDIR}/package-lifecycle.md ${STAGEDIR}${DOCSDIR}/package-lifecycle.md

.include <bsd.port.post.mk>
