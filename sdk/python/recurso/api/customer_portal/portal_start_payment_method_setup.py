from http import HTTPStatus
from typing import Any

import httpx

from ... import errors
from ...client import AuthenticatedClient, Client
from ...models.error import Error
from ...models.portal_start_payment_method_setup_response_200 import PortalStartPaymentMethodSetupResponse200
from ...types import Response


def _get_kwargs() -> dict[str, Any]:

    _kwargs: dict[str, Any] = {
        "method": "post",
        "url": "/portal/api/payment-method/setup-intent",
    }

    return _kwargs


def _parse_response(
    *, client: AuthenticatedClient | Client, response: httpx.Response
) -> Error | PortalStartPaymentMethodSetupResponse200 | None:
    if response.status_code == 200:
        response_200 = PortalStartPaymentMethodSetupResponse200.from_dict(response.json())

        return response_200

    if response.status_code == 401:
        response_401 = Error.from_dict(response.json())

        return response_401

    if response.status_code == 503:
        response_503 = Error.from_dict(response.json())

        return response_503

    if client.raise_on_unexpected_status:
        raise errors.UnexpectedStatus(response.status_code, response.content)
    else:
        return None


def _build_response(
    *, client: AuthenticatedClient | Client, response: httpx.Response
) -> Response[Error | PortalStartPaymentMethodSetupResponse200]:
    return Response(
        status_code=HTTPStatus(response.status_code),
        content=response.content,
        headers=response.headers,
        parsed=_parse_response(client=client, response=response),
    )


def sync_detailed(
    *,
    client: AuthenticatedClient,
) -> Response[Error | PortalStartPaymentMethodSetupResponse200]:
    """Start a card update (Stripe SetupIntent)

     Creates a Stripe SetupIntent for the authenticated portal customer and returns the client secret the
    Payment Element confirms. Card data goes browser→Stripe directly — no PAN ever reaches Recurso (PCI
    SAQ-A). Reports 503 on deployments without Stripe.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Error | PortalStartPaymentMethodSetupResponse200]
    """

    kwargs = _get_kwargs()

    response = client.get_httpx_client().request(
        **kwargs,
    )

    return _build_response(client=client, response=response)


def sync(
    *,
    client: AuthenticatedClient,
) -> Error | PortalStartPaymentMethodSetupResponse200 | None:
    """Start a card update (Stripe SetupIntent)

     Creates a Stripe SetupIntent for the authenticated portal customer and returns the client secret the
    Payment Element confirms. Card data goes browser→Stripe directly — no PAN ever reaches Recurso (PCI
    SAQ-A). Reports 503 on deployments without Stripe.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Error | PortalStartPaymentMethodSetupResponse200
    """

    return sync_detailed(
        client=client,
    ).parsed


async def asyncio_detailed(
    *,
    client: AuthenticatedClient,
) -> Response[Error | PortalStartPaymentMethodSetupResponse200]:
    """Start a card update (Stripe SetupIntent)

     Creates a Stripe SetupIntent for the authenticated portal customer and returns the client secret the
    Payment Element confirms. Card data goes browser→Stripe directly — no PAN ever reaches Recurso (PCI
    SAQ-A). Reports 503 on deployments without Stripe.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Error | PortalStartPaymentMethodSetupResponse200]
    """

    kwargs = _get_kwargs()

    response = await client.get_async_httpx_client().request(**kwargs)

    return _build_response(client=client, response=response)


async def asyncio(
    *,
    client: AuthenticatedClient,
) -> Error | PortalStartPaymentMethodSetupResponse200 | None:
    """Start a card update (Stripe SetupIntent)

     Creates a Stripe SetupIntent for the authenticated portal customer and returns the client secret the
    Payment Element confirms. Card data goes browser→Stripe directly — no PAN ever reaches Recurso (PCI
    SAQ-A). Reports 503 on deployments without Stripe.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Error | PortalStartPaymentMethodSetupResponse200
    """

    return (
        await asyncio_detailed(
            client=client,
        )
    ).parsed
