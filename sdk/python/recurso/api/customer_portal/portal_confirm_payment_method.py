from http import HTTPStatus
from typing import Any

import httpx

from ... import errors
from ...client import AuthenticatedClient, Client
from ...models.error import Error
from ...models.portal_confirm_payment_method_body import PortalConfirmPaymentMethodBody
from ...models.portal_confirm_payment_method_response_200 import PortalConfirmPaymentMethodResponse200
from ...types import Response


def _get_kwargs(
    *,
    body: PortalConfirmPaymentMethodBody,
) -> dict[str, Any]:
    headers: dict[str, Any] = {}

    _kwargs: dict[str, Any] = {
        "method": "post",
        "url": "/portal/api/payment-method/confirm",
    }

    _kwargs["json"] = body.to_dict()

    headers["Content-Type"] = "application/json"

    _kwargs["headers"] = headers
    return _kwargs


def _parse_response(
    *, client: AuthenticatedClient | Client, response: httpx.Response
) -> Error | PortalConfirmPaymentMethodResponse200 | None:
    if response.status_code == 200:
        response_200 = PortalConfirmPaymentMethodResponse200.from_dict(response.json())

        return response_200

    if response.status_code == 400:
        response_400 = Error.from_dict(response.json())

        return response_400

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
) -> Response[Error | PortalConfirmPaymentMethodResponse200]:
    return Response(
        status_code=HTTPStatus(response.status_code),
        content=response.content,
        headers=response.headers,
        parsed=_parse_response(client=client, response=response),
    )


def sync_detailed(
    *,
    client: AuthenticatedClient,
    body: PortalConfirmPaymentMethodBody,
) -> Response[Error | PortalConfirmPaymentMethodResponse200]:
    """Finalize a confirmed SetupIntent

     Verifies the SetupIntent succeeded and that its metadata customer_id matches the authenticated
    portal customer, then records the saved card as the customer's default payment method. Called after
    `stripe.confirmSetup` — inline on success, or from the `?setup_intent=` return param after a
    3DS/bank redirect.

    Args:
        body (PortalConfirmPaymentMethodBody):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Error | PortalConfirmPaymentMethodResponse200]
    """

    kwargs = _get_kwargs(
        body=body,
    )

    response = client.get_httpx_client().request(
        **kwargs,
    )

    return _build_response(client=client, response=response)


def sync(
    *,
    client: AuthenticatedClient,
    body: PortalConfirmPaymentMethodBody,
) -> Error | PortalConfirmPaymentMethodResponse200 | None:
    """Finalize a confirmed SetupIntent

     Verifies the SetupIntent succeeded and that its metadata customer_id matches the authenticated
    portal customer, then records the saved card as the customer's default payment method. Called after
    `stripe.confirmSetup` — inline on success, or from the `?setup_intent=` return param after a
    3DS/bank redirect.

    Args:
        body (PortalConfirmPaymentMethodBody):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Error | PortalConfirmPaymentMethodResponse200
    """

    return sync_detailed(
        client=client,
        body=body,
    ).parsed


async def asyncio_detailed(
    *,
    client: AuthenticatedClient,
    body: PortalConfirmPaymentMethodBody,
) -> Response[Error | PortalConfirmPaymentMethodResponse200]:
    """Finalize a confirmed SetupIntent

     Verifies the SetupIntent succeeded and that its metadata customer_id matches the authenticated
    portal customer, then records the saved card as the customer's default payment method. Called after
    `stripe.confirmSetup` — inline on success, or from the `?setup_intent=` return param after a
    3DS/bank redirect.

    Args:
        body (PortalConfirmPaymentMethodBody):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Error | PortalConfirmPaymentMethodResponse200]
    """

    kwargs = _get_kwargs(
        body=body,
    )

    response = await client.get_async_httpx_client().request(**kwargs)

    return _build_response(client=client, response=response)


async def asyncio(
    *,
    client: AuthenticatedClient,
    body: PortalConfirmPaymentMethodBody,
) -> Error | PortalConfirmPaymentMethodResponse200 | None:
    """Finalize a confirmed SetupIntent

     Verifies the SetupIntent succeeded and that its metadata customer_id matches the authenticated
    portal customer, then records the saved card as the customer's default payment method. Called after
    `stripe.confirmSetup` — inline on success, or from the `?setup_intent=` return param after a
    3DS/bank redirect.

    Args:
        body (PortalConfirmPaymentMethodBody):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Error | PortalConfirmPaymentMethodResponse200
    """

    return (
        await asyncio_detailed(
            client=client,
            body=body,
        )
    ).parsed
